<?php declare(strict_types=1);
namespace Nais\Device\Approval\Controllers;

use Nais\Device\Approval\Session;
use Nais\Device\Approval\Session\User;
use NAVIT\AzureAd\ApiClient;
use NAVIT\AzureAd\Models\Group;
use PHPUnit\Framework\TestCase;
use Psr\Http\Message\ServerRequestInterface as Request;
use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\StreamInterface;
use RuntimeException;

/**
 * @coversDefaultClass Nais\Device\Approval\Controllers\MembershipController
 */
class MembershipControllerTest extends TestCase {
    /**
     * @covers ::__construct
     * @covers ::toggle
     */
    public function testRespondsWithErrorOnMissingUser() : void {
        $controller = new MembershipController(
            $this->createConfiguredMock(Session::class, ['getUser' => null]),
            $this->createMock(ApiClient::class),
            'access-group'
        );

        $body = $this->createMock(StreamInterface::class);
        $body
            ->expects($this->once())
            ->method('write')
            ->with('{"error":"Invalid session"}');

        $response2 = $this->createConfiguredMock(Response::class, ['getBody' => $body]);
        $response2
            ->expects($this->once())
            ->method('withStatus')
            ->with(400)
            ->willReturn($this->createMock(Response::class));

        $response1 = $this->createMock(Response::class);
        $response1
            ->expects($this->once())
            ->method('withHeader')
            ->with('Content-Type', 'application/json')
            ->willReturn($response2);

        $controller->toggle(
            $this->createMock(Request::class),
            $response1
        );
    }

    /**
     * @covers ::toggle
     */
    public function testThrowsExceptionWhenUnableToFetchUserGroups() : void {
        $apiClient = $this->createMock(ApiClient::class);
        $apiClient
            ->expects($this->once())
            ->method('getUserGroups')
            ->with('user-id')
            ->willThrowException(new RuntimeException('some error', 500));

        $controller = new MembershipController(
            $this->createConfiguredMock(Session::class, [
                'getUser' => $this->createConfiguredMock(User::class, [
                    'getObjectId' => 'user-id',
                ]),
            ]),
            $apiClient,
            'access-group'
        );

        $body = $this->createMock(StreamInterface::class);
        $body
            ->expects($this->once())
            ->method('write')
            ->with('{"error":"Unable to fetch user groups"}');

        $response2 = $this->createConfiguredMock(Response::class, [
            'getBody' => $body,
        ]);
        $response2
            ->expects($this->once())
            ->method('withStatus')
            ->with(400)
            ->willReturn($this->createMock(Response::class));

        $response1 = $this->createMock(Response::class);
        $response1
            ->expects($this->once())
            ->method('withHeader')
            ->willReturn($response2);

        $controller->toggle(
            $this->createMock(Request::class),
            $response1
        );
    }

    /**
     * @covers ::toggle
     */
    public function testThrowsExceptionWhenTogglingFails() : void {
        $apiClient = $this->createMock(ApiClient::class);
        $apiClient
            ->expects($this->once())
            ->method('getUserGroups')
            ->with('user-id')
            ->willReturn([]);

        $apiClient
            ->expects($this->once())
            ->method('addUserToGroup')
            ->with('user-id', 'access-group')
            ->willThrowException(new RuntimeException('some error', 400));

        $controller = new MembershipController(
            $this->createConfiguredMock(Session::class, [
                'getUser' => $this->createConfiguredMock(User::class, [
                    'getObjectId' => 'user-id',
                ]),
            ]),
            $apiClient,
            'access-group'
        );

        $body = $this->createMock(StreamInterface::class);
        $body
            ->expects($this->once())
            ->method('write')
            ->with('{"error":"Unable to toggle group membership"}');

        $response2 = $this->createConfiguredMock(Response::class, [
            'getBody' => $body,
        ]);
        $response2
            ->expects($this->once())
            ->method('withStatus')
            ->with(400)
            ->willReturn($this->createMock(Response::class));

        $response1 = $this->createMock(Response::class);
        $response1
            ->expects($this->once())
            ->method('withHeader')
            ->willReturn($response2);

        $controller->toggle(
            $this->createMock(Request::class),
            $response1
        );
    }

    /**
     * @covers ::toggle
     */
    public function testCanRemoveUserFromGroup() : void {
        $apiClient = $this->createMock(ApiClient::class);
        $apiClient
            ->expects($this->once())
            ->method('getUserGroups')
            ->with('user-id')
            ->willReturn([
                $this->createConfiguredMock(Group::class, ['getId' => 'group-1']),
                $this->createConfiguredMock(Group::class, ['getId' => 'access-group']),
                $this->createConfiguredMock(Group::class, ['getId' => 'group-2']),
            ]);

        $apiClient
            ->expects($this->once())
            ->method('removeUserFromGroup')
            ->with('user-id', 'access-group');

        $controller = new MembershipController(
            $this->createConfiguredMock(Session::class, [
                'getUser' => $this->createConfiguredMock(User::class, [
                    'getObjectId' => 'user-id',
                ]),
            ]),
            $apiClient,
            'access-group'
        );

        $body = $this->createMock(StreamInterface::class);
        $body
            ->expects($this->once())
            ->method('write')
            ->with('{"success":true,"hasAccepted":false}');

        $response2 = $this->createConfiguredMock(Response::class, [
            'getBody' => $body,
        ]);
        $response2
            ->expects($this->once())
            ->method('withStatus')
            ->with(200)
            ->willReturn($this->createMock(Response::class));

        $response1 = $this->createMock(Response::class);
        $response1
            ->expects($this->once())
            ->method('withHeader')
            ->willReturn($response2);

        $controller->toggle(
            $this->createMock(Request::class),
            $response1
        );
    }

    /**
     * @covers ::toggle
     */
    public function testCanAddUserToGroup() : void {
        $apiClient = $this->createMock(ApiClient::class);
        $apiClient
            ->expects($this->once())
            ->method('getUserGroups')
            ->with('user-id')
            ->willReturn([
                $this->createConfiguredMock(Group::class, ['getId' => 'group-1']),
                $this->createConfiguredMock(Group::class, ['getId' => 'group-2']),
            ]);

        $apiClient
            ->expects($this->once())
            ->method('addUserToGroup')
            ->with('user-id', 'access-group');

        $controller = new MembershipController(
            $this->createConfiguredMock(Session::class, [
                'getUser' => $this->createConfiguredMock(User::class, [
                    'getObjectId' => 'user-id',
                ]),
            ]),
            $apiClient,
            'access-group'
        );

        $body = $this->createMock(StreamInterface::class);
        $body
            ->expects($this->once())
            ->method('write')
            ->with('{"success":true,"hasAccepted":true}');

        $response2 = $this->createConfiguredMock(Response::class, [
            'getBody' => $body,
        ]);
        $response2
            ->expects($this->once())
            ->method('withStatus')
            ->with(200)
            ->willReturn($this->createMock(Response::class));

        $response1 = $this->createMock(Response::class);
        $response1
            ->expects($this->once())
            ->method('withHeader')
            ->willReturn($response2);

        $controller->toggle(
            $this->createMock(Request::class),
            $response1
        );
    }
}