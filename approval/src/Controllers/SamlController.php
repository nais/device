<?php declare(strict_types=1);
namespace Nais\Device\Approval\Controllers;

use Nais\Device\Approval\Session;
use Nais\Device\Approval\Session\User;
use Psr\Http\Message\ServerRequestInterface as Request;
use Psr\Http\Message\ResponseInterface as Response;
use SimpleXMLElement;

class SamlController {
    private Session $session;
    private string $certificate;
    private string $logoutUrl;

    public function __construct(Session $session, string $certificate, string $logoutUrl) {
        $this->session     = $session;
        $this->certificate = $certificate;
        $this->logoutUrl   = $logoutUrl;
    }

    public function acs(Request $request, Response $response) : Response {
        if ($this->session->hasUser()) {
            $response->getBody()->write('User has already been authenticated');
            return $response->withStatus(400);
        }

        /** @var array{SAMLResponse: ?string} */
        $params = $request->getParsedBody();

        if (empty($params['SAMLResponse'])) {
            $response->getBody()->write('Missing SAML response');
            return $response->withStatus(400);
        }

        $decoded = base64_decode($params['SAMLResponse'], true);

        if (false === $decoded) {
            $response->getBody()->write('Value is not properly base64 encoded');
            return $response->withStatus(400);
        }

        $xml = simplexml_load_string($decoded, 'SimpleXMLElement', LIBXML_NOERROR);

        if (false === $xml) {
            $response->getBody()->write('Value is not proper XML');
            return $response->withStatus(400);
        }

        $xml->registerXPathNamespace('samlp', 'urn:oasis:names:tc:SAML:2.0:protocol');
        $xml->registerXPathNamespace('a',     'urn:oasis:names:tc:SAML:2.0:assertion');
        $xml->registerXPathNamespace('s',     'http://www.w3.org/2000/09/xmldsig#');

        $cert = $xml->xpath('/samlp:Response/a:Assertion/s:Signature/s:KeyInfo/s:X509Data/s:X509Certificate');

        if (empty($cert)) {
            $response->getBody()->write('Missing certificate');
            return $response->withStatus(400);
        }

        $cert = (string) $cert[0];

        openssl_x509_export("-----BEGIN CERTIFICATE-----\n" . $cert . "\n-----END CERTIFICATE-----", $incomingCert);

        if (openssl_x509_fingerprint($incomingCert) !== openssl_x509_fingerprint($this->certificate)) {
            $response->getBody()->write('Invalid certificate');
            return $response->withStatus(400);
        }

        /** @var SimpleXMLElement[] */
        $elems = $xml->xpath('//a:Attribute[@Name="http://schemas.microsoft.com/identity/claims/objectidentifier"]/a:AttributeValue');
        $objectId = $elems[0] ?? null;

        if (null === $objectId) {
            $response->getBody()->write('Missing objectidentifier claim');
            return $response->withStatus(400);
        }

        /** @var SimpleXMLElement[] */
        $elems = $xml->xpath('//a:Attribute[@Name="http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname"]/a:AttributeValue');
        $givenName = $elems[0] ?? null;

        if (null === $givenName) {
            $response->getBody()->write('Missing givenName claim');
            return $response->withStatus(400);
        }

        $this->session->setUser(new User(
            (string) $objectId,
            (string) $givenName
        ));

        return $response
            ->withHeader('Location', '/')
            ->withStatus(302);
    }

    public function logout(Request $request, Response $response) : Response {
        $this->session->destroy();

        return $response
            ->withHeader('Location', $this->logoutUrl)
            ->withStatus(302);
    }
}