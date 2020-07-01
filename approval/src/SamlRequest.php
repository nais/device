<?php declare(strict_types=1);
namespace Nais\Device\Approval;

use DateTime;
use DateTimeZone;

class SamlRequest {
    private string $issuer;

    /**
     * Class constructor
     *
     * @param string $issuer
     */
    public function __construct(string $issuer) {
        $this->issuer = $issuer;
    }

    /**
     * Render as string
     *
     * @return string
     */
    public function __toString() : string {
        $samlRequest = <<<SAMLRequest
<samlp:AuthnRequest
    xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol"
    xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion"
    ID="id_%s"
    Version="2.0"
    IssueInstant="%s"
>
    <saml:Issuer>%s</saml:Issuer>
</samlp:AuthnRequest>
SAMLRequest;

        $samlRequest = sprintf(
            $samlRequest,
            uniqid(),
            (new DateTime('now', new DateTimeZone('UTC')))->format('Y-m-d\TH:i:s\Z'),
            htmlspecialchars($this->issuer, ENT_QUOTES)
        );

        return base64_encode((string) gzdeflate($samlRequest));
    }
}