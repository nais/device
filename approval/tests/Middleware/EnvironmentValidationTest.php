<?php declare(strict_types=1);
namespace Nais\Device\Approval\Middleware;

use PHPUnit\Framework\TestCase;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Psr\Http\Server\RequestHandlerInterface;
use RuntimeException;

/**
 * @coversDefaultClass Nais\Device\Approval\Middleware\EnvironmentValidation
 */
class EnvironmentValidationTest extends TestCase {
    /**
     * @return array<string, array{0: array<string, string>, 1: string}>
     */
    public function getEnvVars() : array {
        return [
            'no vars' => [
                [],
                'Missing required environment variable(s): LOGIN_URL, ACCESS_GROUP, AAD_CLIENT_ID, AAD_CLIENT_SECRET, SAML_CERT',
            ],
            'missing' => [
                [
                    'LOGIN_URL'     => 'some url',
                    'AAD_CLIENT_ID' => 'some id',
                    'SAML_CERT'     => 'some cert',
                ],
                'Missing required environment variable(s): ACCESS_GROUP, AAD_CLIENT_SECRET',
            ],
        ];
    }

    /**
     * @dataProvider getEnvVars
     * @covers ::__construct
     * @covers ::__invoke
     * @param array<string, string> $vars
     * @param string $error
     */
    public function testFailsOnMissingValue(array $vars, string $error) : void {
        $this->expectExceptionObject(new RuntimeException($error));
        (new EnvironmentValidation($vars))(
            $this->createMock(ServerRequestInterface::class),
            $this->createMock(RequestHandlerInterface::class)
        );
    }

    /**
     * @covers ::__invoke
     */
    public function testHandleResponseOnSuccess() : void {
        $request  = $this->createMock(ServerRequestInterface::class);
        $response = $this->createMock(ResponseInterface::class);

        $handler  = $this->createMock(RequestHandlerInterface::class);
        $handler
            ->expects($this->once())
            ->method('handle')
            ->with($request)
            ->willReturn($response);

        $middleware = new EnvironmentValidation([
            'LOGIN_URL'         => 'some url',
            'ACCESS_GROUP'      => 'some group',
            'AAD_CLIENT_ID'     => 'some id',
            'AAD_CLIENT_SECRET' => 'some secret',
            'SAML_CERT'         => 'some cert',
        ]);
        $this->assertSame($response, $middleware($request, $handler));
    }
}