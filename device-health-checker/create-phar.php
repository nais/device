<?php declare(strict_types=1);
function usage(string $extra = null, int $exitCode = 1) : void {
    echo sprintf('usage: php -d phar.readonly=off %s <script> <target-path>', $_SERVER['argv'][0]) . PHP_EOL;

    if (null !== $extra) {
        echo PHP_EOL . $extra . PHP_EOL;
    }

    exit($exitCode);
}

if (3 !== $_SERVER['argc']) {
    usage('Incorrect parameter count');
}

// Path to the script we want to package
$script = $_SERVER['argv'][1];

// Directory to store the archive
$targetPath = $_SERVER['argv'][2];

if (!is_file($script)) {
    usage(sprintf('Script file does not exist: %s', $script));
} else if (ini_get('phar.readonly')) {
    usage('Set the phar.readonly PHP ini setting to off');
} else if (is_dir($targetPath) && !is_writable($targetPath)) {
    usage(sprintf('Target path exists but is not writable: %s', $targetPath));
} else if (!is_dir($targetPath) && !mkdir($targetPath, 0777, true)) {
    usage(sprintf('Unable to create target path: %s', $targetPath));
}

$archiveName = sprintf('%s/%s', $_SERVER['argv'][2], preg_replace('/\.php$/', '.phar', basename($script)));

if (file_exists($archiveName) && !unlink($archiveName)) {
    throw new RuntimeException(sprintf(
        'Unable to remove existing archive: %s. Please remove manually before running this command again',
        $archiveName
    ));
}

// Files / dirs to add to the archive
$whitelist = [
    'src',
    'vendor',
    'checks-config.php',
    basename($script),
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
require "phar://%s/%s";
__HALT_COMPILER();
STUB;

$phar = new Phar($archiveName);
$phar->setAlias(basename($archiveName));
$phar->buildFromIterator($iterator, __DIR__);
$phar->setStub(sprintf($stubContents, basename($archiveName), basename($script)));
$phar->compressFiles(Phar::GZ);
