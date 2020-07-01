<?php declare(strict_types=1);
namespace Nais\Device\Approval\Controllers;

use Nais\Device\Approval\Session;
use Nais\Device\Approval\SamlRequest;
use NAVIT\AzureAd\ApiClient;
use NAVIT\AzureAd\Models\Group;
use Slim\Views\Twig;
use Psr\Http\Message\ServerRequestInterface as Request;
use Psr\Http\Message\ResponseInterface as Response;
use RuntimeException;

class IndexController {
    private ApiClient $apiClient;
    private Twig $view;
    private Session $session;
    private string $loginUrl;
    private string $entityId;
    private string $accessGroup;

    public function __construct(ApiClient $apiClient, Twig $view, Session $session, string $loginUrl, string $entityId, string $accessGroup) {
        $this->apiClient   = $apiClient;
        $this->view        = $view;
        $this->session     = $session;
        $this->loginUrl    = $loginUrl;
        $this->entityId    = $entityId;
        $this->accessGroup = $accessGroup;
    }

    public function index(Request $request, Response $response) : Response {
        $user = $this->session->getUser();

        if (null === $user) {
            return $response
                ->withHeader('Location', sprintf(
                    '%s?SAMLRequest=%s',
                    $this->loginUrl,
                    urlencode((string) new SamlRequest($this->entityId))
                ))
                ->withStatus(302);
        }

        try {
            $groups = array_filter($this->apiClient->getUserGroups($user->getObjectId()), function(Group $group) : bool {
                return $group->getId() === $this->accessGroup;
            });
        } catch (RuntimeException $e) {
            throw new RuntimeException('Unable to fetch user groups', (int) $e->getCode(), $e);
        }

        $token = uniqid('', true);
        $this->session->setPostToken($token);

        return $this->view->render($response, 'index.html', [
            'user'        => $user,
            'hasAccepted' => 0 !== count($groups),
            'token'       => $token,
        ]);
    }
}