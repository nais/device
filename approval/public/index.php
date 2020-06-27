<?php declare(strict_types=1);
namespace Nais\Device\Approval;

use NAVIT\AzureAd\ApiClient;
use DI\Container;
use Slim\Factory\AppFactory;
use Slim\Views\Twig;
use Slim\Views\TwigMiddleware;
use Nais\Device\Approval\Controllers\IndexController;
use Nais\Device\Approval\Controllers\MembershipController;
use Nais\Device\Approval\Controllers\SamlController;
use Psr\Container\ContainerInterface;
use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;
use Psr\Http\Server\RequestHandlerInterface as RequestHandler;
use RuntimeException;
use Throwable;

require __DIR__ . '/../vendor/autoload.php';

define('ISSUER_ENTITY_ID',  'naisdevice-approval');
define('LOGIN_URL',         'https://login.microsoftonline.com/62366534-1ec3-4962-8869-9b5535279d0b/saml2');
define('LOGOUT_URL',        'https://login.microsoftonline.com/common/wsfederation?wa=wsignout1.0');
define('ACCESS_GROUP',      'ffd89425-c75c-4618-b5ab-67149ddbbc2d');
define('AAD_CLIENT_ID',     '954ef047-2f2b-49b8-ab8b-6b86f1fe6982');
define('AAD_CLIENT_SECRET', (string) getenv('AAD_CLIENT_SECRET'));
define('SAML_CERT',         (string) getenv('SAML_CERT'));
define('DEBUG',             '1' === getenv('DEBUG'));

// Create and populate container
$container = new Container();
$container->set(Twig::class, fn() : Twig => Twig::create(__DIR__ . '/../templates'));
$container->set(Session::class, fn() : Session => (new Session())->start() );
$container->set(ApiClient::class, fn() => new ApiClient(AAD_CLIENT_ID, AAD_CLIENT_SECRET, 'nav.no'));
$container->set(IndexController::class, function(ContainerInterface $c) : IndexController {
    /** @var ApiClient */
    $apiClient = $c->get(ApiClient::class);

    /** @var Twig */
    $twig = $c->get(Twig::class);

    /** @var Session */
    $session = $c->get(Session::class);

    return  new IndexController($apiClient, $twig, $session, LOGIN_URL, ISSUER_ENTITY_ID, ACCESS_GROUP);
});
$container->set(SamlController::class, function(ContainerInterface $c) : SamlController {
    /** @var Session */
    $session = $c->get(Session::class);

    return new SamlController($session, SAML_CERT, LOGOUT_URL);
});
$container->set(MembershipController::class, function(ContainerInterface $c) : MembershipController {
    /** @var Session */
    $session = $c->get(Session::class);

    /** @var ApiClient */
    $apiClient = $c->get(ApiClient::class);

    return new MembershipController($session, $apiClient, ACCESS_GROUP);
});

AppFactory::setContainer($container);
$app = AppFactory::create();

// Register middleware
$app->addBodyParsingMiddleware();
$app->add(TwigMiddleware::createFromContainer($app, Twig::class));
$app->add(function(Request $request, RequestHandler $handler) : Response {
    if ('' === AAD_CLIENT_SECRET) {
        throw new RuntimeException('Missing AAD_CLIENT_SECRET environment variable');
    } else if ('' === SAML_CERT) {
        throw new RuntimeException('Missing SAML_CERT environment variable');
    }

    return $handler->handle($request);
});
$app
    ->addErrorMiddleware(DEBUG, true, true)
    ->setDefaultErrorHandler(function(Request $request, Throwable $exception, bool $displayErrorDetails) use ($app) {
        /** @var ContainerInterface */
        $container = $app->getContainer();

        /** @var Twig */
        $twig = $container->get(Twig::class);

        return $twig->render($app->getResponseFactory()->createResponse(500), 'error.html', [
            'errorMessage' => $displayErrorDetails ? $exception->getMessage() : 'An error occurred',
        ]);
    });

// Routes
$app->get('/',                  IndexController::class . ':index');
$app->post('/toggleMembership', MembershipController::class . ':toggle');
$app->post('/saml/acs',         SamlController::class . ':acs');
$app->get('/saml/logout',       SamlController::class . ':logout');
$app->get('/isAlive',           fn(Request $request, Response $response) : Response => $response);
$app->get('/isReady',           fn(Request $request, Response $response) : Response => $response);

// Run the app
$app->run();