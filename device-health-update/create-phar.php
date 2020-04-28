<?php declare(strict_types=1);
function usage(string $extra = null) : void {
    echo sprintf('usage: php -dphar.readonly=off %s <stub-file>', $_SERVER['argv'][0]) . PHP_EOL;

    if (null !== $extra) {
        echo $extra . PHP_EOL;
    }
}

if (empty($_SERVER['argv'][1])) {
    usage();
    exit(1);
}

$stub = realpath($_SERVER['argv'][1]);

if (!is_file($stub)) {
    usage(sprintf('Stub file does not exist: %s', $_SERVER['argv'][1]));
    exit(2);
} else if (ini_get('phar.readonly')) {
    usage('Remember to set the phar.readonly PHP ini setting to off');
    exit(3);
}

$archiveName = preg_replace('/\.php$/', '.phar', $stub);

if (file_exists($archiveName) && !unlink($archiveName)) {
    throw new RuntimeException(sprintf(
        'Unable to remove existing archive: %s. Please remove manually before running this command again',
        $archiveName
    ));
}

$whitelist = [
    'src',
    'vendor',
    basename($stub),
];

$iterator = new RecursiveIteratorIterator(
    new RecursiveCallbackFilterIterator(
        new RecursiveDirectoryIterator(__DIR__, RecursiveDirectoryIterator::SKIP_DOTS),
        function (SplFileInfo $file) use ($whitelist) : bool {
            $name = trim(str_replace(__DIR__, '', (string) $file), '/');

            foreach ($whitelist as $fileToAddToArchive) {
                if (0 === strpos($name, $fileToAddToArchive)) {
                    return true;
                }
            }

            return false;
        }
    )
);

$stubContents = <<<STUB
#!/usr/bin/env php
<?php
Phar::mapPhar();
require "phar://{basename($archiveName)}/{basename($stub)}";
__HALT_COMPILER();
STUB;

$phar = new Phar($archiveName);
$phar->setAlias(basename($archiveName));
$phar->buildFromIterator($iterator, __DIR__);
$phar->setStub($stubContents);
$phar->compressFiles(Phar::GZ);