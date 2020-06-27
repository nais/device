<?php declare(strict_types=1);
namespace Nais\Device\Approval\Controllers;

use Nais\Device\Approval\Session;
use Nais\Device\Approval\Session\User;
use NAVIT\AzureAd\ApiClient;
use NAVIT\AzureAd\Models\Group;
use PHPUnit\Framework\TestCase;
use Psr\Http\Message\ServerRequestInterface as Request;
use Psr\Http\Message\ResponseInterface as Response;
use RuntimeException;
use Slim\Views\Twig;

/**
 * @coversDefaultClass Nais\Device\Approval\Controllers\IndexController
 */
class IndexControllerTest extends TestCase {
    /**
     * @covers ::__construct
     * @covers ::index
     */
    public function testRedirectsOnMissingUser() : void {
        $controller = new IndexController(
            $this->createMock(ApiClient::class),
            $this->createMock(Twig::class),
            $this->createConfiguredMock(Session::class, [
                'getUser' => null,
            ]),
            'loginurl', 'entityid', 'access-group'
        );

        $locationResponse = $this->createMock(Response::class);
        $locationResponse
            ->expects($this->once())
            ->method('withStatus')
            ->with(302)
            ->willReturn($this->createMock(Response::class));

        $response = $this->createMock(Response::class);
        $response
            ->expects($this->once())
            ->method('withHeader')
            ->with('Location', $this->callback(fn(string $url) : bool => 0 === strpos($url, 'loginurl?SAMLRequest=')))
            ->willReturn($locationResponse);

        $controller->index(
            $this->createMock(Request::class),
            $response
        );
    }

    /**
     * @covers ::index
     */
    public function testThrowsExceptionWhenUnableToGetUserGroups() : void {
        $apiClient = $this->createMock(ApiClient::class);
        $apiClient
            ->expects($this->once())
            ->method('getUserGroups')
            ->with('user-id')
            ->willThrowException(new RuntimeException('Some error occurred', 400));

        $controller = new IndexController(
            $apiClient,
            $this->createMock(Twig::class),
            $this->createConfiguredMock(Session::class, [
                'getUser' => $this->createConfiguredMock(User::class, [
                    'getObjectId' => 'user-id'
                ]),
            ]),
            'loginurl', 'entityid', 'access-group'
        );

        $this->expectExceptionObject(new RuntimeException('Unable to fetch user groups', 400));
        $controller->index(
            $this->createMock(Request::class),
            $this->createMock(Response::class)
        );
    }

    /**
     * @return array<string, array{0: Group[], 1: string, 2: bool}>
     */
    public function getUserGroups() : array {
        return [
            'no groups' => [
                [],
                'access-group',
                false
            ],
            'has approval group' => [
                [
                    $this->createConfiguredMock(Group::class, ['getId' => 'group-1']),
                    $this->createConfiguredMock(Group::class, ['getId' => 'access-group']),
                    $this->createConfiguredMock(Group::class, ['getId' => 'group-2']),
                ],
                'access-group',
                true
            ],
            'does not have approval group' => [
                [
                    $this->createConfiguredMock(Group::class, ['getId' => 'group-1']),
                    $this->createConfiguredMock(Group::class, ['getId' => 'group-2']),
                ],
                'access-group',
                false
            ],
        ];
    }

    /**
     * @dataProvider getUserGroups
     * @covers ::index
     * @param Group[] $groups
     * @param string $accessGroup
     * @param bool $hasAccepted
     */
    public function testCanRenderViewWithCorrectVariables(array $groups, string $accessGroup, bool $hasAccepted) : void {
        $apiClient = $this->createMock(ApiClient::class);
        $apiClient
            ->expects($this->once())
            ->method('getUserGroups')
            ->with('user-id')
            ->willReturn($groups);

        $user = $this->createConfiguredMock(User::class, [
            'getObjectId' => 'user-id'
        ]);

        $view = $this->createMock(Twig::class);
        $view
            ->expects($this->once())
            ->method('render')
            ->with($this->isInstanceOf(Response::class), 'index.html', [
                'user'        => $user,
                'hasAccepted' => $hasAccepted
            ]);

        $controller = new IndexController(
            $apiClient,
            $view,
            $this->createConfiguredMock(Session::class, [
                'getUser' => $user,
            ]),
            'loginurl', 'entityid', $accessGroup
        );

        $controller->index(
            $this->createMock(Request::class),
            $this->createMock(Response::class)
        );
    }
}