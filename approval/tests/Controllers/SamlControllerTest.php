<?php declare(strict_types=1);
namespace Nais\Device\Approval\Controllers;

use Nais\Device\Approval\Session;
use Nais\Device\Approval\Session\User;
use PHPUnit\Framework\TestCase;
use Psr\Http\Message\ServerRequestInterface as Request;
use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\StreamInterface;

/**
 * @coversDefaultClass Nais\Device\Approval\Controllers\SamlController
 */
class SamlControllerTest extends TestCase {
    /**
     * @covers ::__construct
     * @covers ::logout
     */
    public function testCanLogOut() : void {
        $session = $this->createMock(Session::class);
        $session
            ->expects($this->once())
            ->method('destroy');

        $response2 = $this->createMock(Response::class);
        $response2
            ->expects($this->once())
            ->method('withStatus')
            ->with(302)
            ->willReturn($this->createMock(Response::class));

        $response1 = $this->createMock(Response::class);
        $response1
            ->expects($this->once())
            ->method('withHeader')
            ->with('Location', 'logout-url')
            ->willReturn($response2);

        (new SamlController($session, 'cert', 'logout-url'))->logout(
            $this->createMock(Request::class),
            $response1
        );
    }

    /**
     * @covers ::acs
     */
    public function testFailsWhenUserAlreadyExists() : void {
        $body = $this->createMock(StreamInterface::class);
        $body
            ->expects($this->once())
            ->method('write')
            ->with('User has already been authenticated');

        $response1 = $this->createConfiguredMock(Response::class, ['getBody' => $body]);
        $response1
            ->expects($this->once())
            ->method('withStatus')
            ->with(400)
            ->willReturn($this->createMock(Response::class));

        $controller = new SamlController(
            $this->createConfiguredMock(Session::class, ['hasUser' => true]),
            'cert',
            'logout-url'
        );
        $controller->acs(
            $this->createMock(Request::class),
            $response1
        );
    }

    /**
     * @covers ::acs
     */
    public function testFailsWhenRequestIsMissingSamlResponse() : void {
        $body = $this->createMock(StreamInterface::class);
        $body
            ->expects($this->once())
            ->method('write')
            ->with('Missing SAML response');

        $response1 = $this->createConfiguredMock(Response::class, ['getBody' => $body]);
        $response1
            ->expects($this->once())
            ->method('withStatus')
            ->with(400)
            ->willReturn($this->createMock(Response::class));

        $controller = new SamlController($this->createConfiguredMock(Session::class, ['hasUser' => false]), 'cert', 'logout-url');
        $controller->acs(
            $this->createConfiguredMock(Request::class, [
                'getParsedBody' => [],
            ]),
            $response1
        );
    }

    /**
     * @covers ::acs
     */
    public function testFailsWhenSamlResponseIsNotCorrectlyEncoded() : void {
        $body = $this->createMock(StreamInterface::class);
        $body
            ->expects($this->once())
            ->method('write')
            ->with('Value is not properly base64 encoded');

        $response1 = $this->createConfiguredMock(Response::class, ['getBody' => $body]);
        $response1
            ->expects($this->once())
            ->method('withStatus')
            ->with(400)
            ->willReturn($this->createMock(Response::class));

        $controller = new SamlController($this->createConfiguredMock(Session::class, ['hasUser' => false]), 'cert', 'logout-url');
        $controller->acs(
            $this->createConfiguredMock(Request::class, [
                'getParsedBody' => [
                    'SAMLResponse' => '<foobar>'
                ],
            ]),
            $response1
        );
    }

    /**
     * @covers ::acs
     */
    public function testFailsWhenSamlResponseContainsInvalidXml() : void {
        $body = $this->createMock(StreamInterface::class);
        $body
            ->expects($this->once())
            ->method('write')
            ->with('Value is not proper XML');

        $response1 = $this->createConfiguredMock(Response::class, ['getBody' => $body]);
        $response1
            ->expects($this->once())
            ->method('withStatus')
            ->with(400)
            ->willReturn($this->createMock(Response::class));

        $controller = new SamlController($this->createConfiguredMock(Session::class, ['hasUser' => false]), 'cert', 'logout-url');
        $controller->acs(
            $this->createConfiguredMock(Request::class, [
                'getParsedBody' => [
                    'SAMLResponse' => base64_encode('some string')
                ],
            ]),
            $response1
        );
    }

    /**
     * @covers ::acs
     */
    public function testFailsWhenSamlResponseIsMissingCert() : void {
        $body = $this->createMock(StreamInterface::class);
        $body
            ->expects($this->once())
            ->method('write')
            ->with('Missing certificate');

        $response1 = $this->createConfiguredMock(Response::class, ['getBody' => $body]);
        $response1
            ->expects($this->once())
            ->method('withStatus')
            ->with(400)
            ->willReturn($this->createMock(Response::class));

        $controller = new SamlController(
            $this->createConfiguredMock(Session::class, ['hasUser' => false]),
            (string) file_get_contents(__DIR__ . '/../fixtures/example-cert.pem'),
            'logout-url'
        );
        $controller->acs(
            $this->createConfiguredMock(Request::class, [
                'getParsedBody' => [
                    'SAMLResponse' => base64_encode((string) file_get_contents(__DIR__ . '/../fixtures/response-with-missing-cert.xml'))
                ],
            ]),
            $response1
        );
    }

    /**
     * @covers ::acs
     */
    public function testFailsWhenSamlResponseHasTheWrongCert() : void {
        $body = $this->createMock(StreamInterface::class);
        $body
            ->expects($this->once())
            ->method('write')
            ->with('Invalid certificate');

        $response1 = $this->createConfiguredMock(Response::class, ['getBody' => $body]);
        $response1
            ->expects($this->once())
            ->method('withStatus')
            ->with(400)
            ->willReturn($this->createMock(Response::class));

        $controller = new SamlController(
            $this->createConfiguredMock(Session::class, ['hasUser' => false]),
            (string) file_get_contents(__DIR__ . '/../fixtures/example-cert.pem'),
            'logout-url'
        );
        $controller->acs(
            $this->createConfiguredMock(Request::class, [
                'getParsedBody' => [
                    'SAMLResponse' => base64_encode((string) file_get_contents(__DIR__ . '/../fixtures/response-with-invalid-cert.xml'))
                ],
            ]),
            $response1
        );
    }

    /**
     * @covers ::acs
     */
    public function testFailsWhenSamlResponseIsMissingObjectId() : void {
        $body = $this->createMock(StreamInterface::class);
        $body
            ->expects($this->once())
            ->method('write')
            ->with('Missing objectidentifier claim');

        $response1 = $this->createConfiguredMock(Response::class, ['getBody' => $body]);
        $response1
            ->expects($this->once())
            ->method('withStatus')
            ->with(400)
            ->willReturn($this->createMock(Response::class));

        $controller = new SamlController(
            $this->createConfiguredMock(Session::class, ['hasUser' => false]),
            (string) file_get_contents(__DIR__ . '/../fixtures/example-cert.pem'),
            'logout-url'
        );
        $controller->acs(
            $this->createConfiguredMock(Request::class, [
                'getParsedBody' => [
                    'SAMLResponse' => base64_encode((string) file_get_contents(__DIR__ . '/../fixtures/response-with-missing-object-id.xml'))
                ],
            ]),
            $response1
        );
    }

    /**
     * @covers ::acs
     */
    public function testFailsWhenSamlResponseIsMissingGivenNameObjectId() : void {
        $body = $this->createMock(StreamInterface::class);
        $body
            ->expects($this->once())
            ->method('write')
            ->with('Missing givenName claim');

        $response1 = $this->createConfiguredMock(Response::class, ['getBody' => $body]);
        $response1
            ->expects($this->once())
            ->method('withStatus')
            ->with(400)
            ->willReturn($this->createMock(Response::class));

        $controller = new SamlController(
            $this->createConfiguredMock(Session::class, ['hasUser' => false]),
            (string) file_get_contents(__DIR__ . '/../fixtures/example-cert.pem'),
            'logout-url'
        );
        $controller->acs(
            $this->createConfiguredMock(Request::class, [
                'getParsedBody' => [
                    'SAMLResponse' => base64_encode((string) file_get_contents(__DIR__ . '/../fixtures/response-with-missing-given-name.xml'))
                ],
            ]),
            $response1
        );
    }

    /**
     * @covers ::acs
     */
    public function testCanSuccessfullySetUserInSession() : void {
        $response2 = $this->createMock(Response::class);
        $response2
            ->expects($this->once())
            ->method('withStatus')
            ->with(302)
            ->willReturn($this->createMock(Response::class));

        $response1 = $this->createMock(Response::class);
        $response1
            ->expects($this->once())
            ->method('withHeader')
            ->with('Location', '/')
            ->willReturn($response2);

        $session = $this->createConfiguredMock(Session::class, ['hasUser' => false]);
        $session
            ->expects($this->once())
            ->method('setUser')
            ->with($this->callback(
                fn(User $user) : bool => 'user-id' === $user->getObjectId() && 'Givenname' === $user->getName())
            );

        $controller = new SamlController(
            $session,
            (string) file_get_contents(__DIR__ . '/../fixtures/example-cert.pem'),
            'logout-url'
        );
        $controller->acs(
            $this->createConfiguredMock(Request::class, [
                'getParsedBody' => [
                    'SAMLResponse' => base64_encode((string) file_get_contents(__DIR__ . '/../fixtures/response.xml'))
                ],
            ]),
            $response1
        );
    }
}