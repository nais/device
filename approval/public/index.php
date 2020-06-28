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
use Throwable;

require __DIR__ . '/../vendor/autoload.php';

/**
 * Get env var as string
 *
 * @param string $key
 * @return string
 */
function env(string $key) : string {
    return (string) getenv($key);
}

define('DEBUG', '1' === env('DEBUG'));

// Create and populate container
$container = new Container();
$container->set(Twig::class, fn() : Twig => Twig::create(__DIR__ . '/../templates'));
$container->set(Session::class, fn() : Session => (new Session())->start() );
$container->set(ApiClient::class, fn() => new ApiClient(env('AAD_CLIENT_ID'), env('AAD_CLIENT_SECRET'), env('DOMAIN')));
$container->set(IndexController::class, function(ContainerInterface $c) : IndexController {
    /** @var ApiClient */
    $apiClient = $c->get(ApiClient::class);

    /** @var Twig */
    $twig = $c->get(Twig::class);

    /** @var Session */
    $session = $c->get(Session::class);

    return  new IndexController($apiClient, $twig, $session, env('LOGIN_URL'), env('ISSUER_ENTITY_ID'), env('ACCESS_GROUP'));
});
$container->set(SamlController::class, function(ContainerInterface $c) : SamlController {
    /** @var Session */
    $session = $c->get(Session::class);

    return new SamlController($session, env('SAML_CERT'), env('LOGOUT_URL'));
});
$container->set(MembershipController::class, function(ContainerInterface $c) : MembershipController {
    /** @var Session */
    $session = $c->get(Session::class);

    /** @var ApiClient */
    $apiClient = $c->get(ApiClient::class);

    return new MembershipController($session, $apiClient, env('ACCESS_GROUP'));
});

AppFactory::setContainer($container);
$app = AppFactory::create();

// Register middleware
$app->addBodyParsingMiddleware();
$app->add(TwigMiddleware::createFromContainer($app, Twig::class));
$app->add(new Middleware\EnvironmentValidation(getenv()));
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