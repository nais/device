<?php declare(strict_types=1);
namespace Nais\Device\Approval;

use SimpleXMLElement;
use PHPUnit\Framework\TestCase;

/**
 * @coversDefaultClass Nais\Device\Approval\SamlRequest
 */
class SamlRequestTest extends TestCase {
    /**
     * @covers ::__construct
     * @covers ::__toString
     */
    public function testCanPresentAsString() : void {
        /** @var SimpleXMLElement */
        $request = simplexml_load_string((string) gzinflate((string) base64_decode((string) new SamlRequest('some-issuer'), true)), 'SimpleXMLElement', 0, 'samlp');
        $request->registerXPathNamespace('saml', 'urn:oasis:names:tc:SAML:2.0:assertion');

        /** @var array<SimpleXMLElement> */
        $elems = $request->xpath('/samlp:AuthnRequest/saml:Issuer');

        $this->assertSame('some-issuer', (string) $elems[0]);
    }
}