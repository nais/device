<?php declare(strict_types=1);
namespace Nais\Device\Approval\Controllers;

use Nais\Device\Approval\Session;
use NAVIT\AzureAd\ApiClient;
use NAVIT\AzureAd\Models\Group;
use Psr\Http\Message\ServerRequestInterface as Request;
use Psr\Http\Message\ResponseInterface as Response;
use RuntimeException;

class MembershipController {
    /** @var Session */
    private $session;

    /** @var ApiClient */
    private $apiClient;

    /** @var string */
    private $accessGroup;

    public function __construct(Session $session, ApiClient $apiClient, string $accessGroup) {
        $this->session     = $session;
        $this->apiClient   = $apiClient;
        $this->accessGroup = $accessGroup;
    }

    public function toggle(Request $request, Response $response) : Response {
        $response         = $response->withHeader('Content-Type', 'application/json');
        $user             = $this->session->getUser();
        $sessionToken     = $this->session->getPostToken();

        /** @var array{token: ?string} */
        $post             = $request->getParsedBody();
        $tokenFromRequest = $post['token'] ?? null;

        if (null === $user) {
            $response->getBody()->write((string) json_encode(['error' => 'Invalid session']));
            return $response->withStatus(400);
        } else if (null === $sessionToken) {
            $response->getBody()->write((string) json_encode(['error' => 'Missing session token']));
            return $response->withStatus(400);
        } else if ($sessionToken !== $tokenFromRequest) {
            $response->getBody()->write((string) json_encode(['error' => 'Incorrect session token']));
            return $response->withStatus(400);
        }

        try {
            $groups = array_filter($this->apiClient->getUserGroups($user->getObjectId()), function(Group $group) : bool {
                return $group->getId() === $this->accessGroup;
            });
        } catch (RuntimeException $e) {
            $response->getBody()->write((string) json_encode(['error' => 'Unable to fetch user groups']));
            return $response->withStatus(400);
        }

        $hasAccepted = 0 !== count($groups);

        try {
            if ($hasAccepted) {
                $this->apiClient->removeUserFromGroup($user->getObjectId(), $this->accessGroup);
            } else {
                $this->apiClient->addUserToGroup($user->getObjectId(), $this->accessGroup);
            }
        } catch (RuntimeException $e) {
            $response->getBody()->write((string) json_encode(['error' => 'Unable to toggle group membership']));
            return $response->withStatus(400);
        }

        $response->getBody()->write((string) json_encode(['success' => true, 'hasAccepted' => !$hasAccepted]));
        return $response->withStatus(200);
    }
}