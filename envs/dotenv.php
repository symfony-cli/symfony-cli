<?php

// Run by the Symfony CLI with the project's Symfony Dotenv component; outputs
// SYMFONY_DOTENV_VARS followed by the name and value of each listed variable,
// NUL-separated. The syntax must stay compatible with PHP 5.5.

$vendorDir = $argv[1];
$path = $argv[2];

spl_autoload_register(function ($class) use ($vendorDir) {
    $namespaces = array(
        'Symfony\\Component\\Dotenv\\' => '/symfony/dotenv/',
        'Symfony\\Component\\Process\\' => '/symfony/process/',
    );
    foreach ($namespaces as $namespace => $dir) {
        if (0 === strpos($class, $namespace)) {
            $file = $vendorDir.$dir.strtr(substr($class, strlen($namespace)), '\\', '/').'.php';
            if (is_file($file)) {
                require $file;
            }

            return;
        }
    }
});

try {
    $dotenv = new Symfony\Component\Dotenv\Dotenv();
    if (method_exists($dotenv, 'bootEnv')) {
        $dotenv->bootEnv($path);
    } elseif (method_exists($dotenv, 'loadEnv')) {
        $dotenv->loadEnv($path);
    } elseif (is_file($path)) {
        $dotenv->load($path);
    }
} catch (Exception $e) {
    fwrite(STDERR, $e->getMessage());
    exit(1);
}

$listed = isset($_ENV['SYMFONY_DOTENV_VARS']) ? $_ENV['SYMFONY_DOTENV_VARS'] : (isset($_SERVER['SYMFONY_DOTENV_VARS']) ? $_SERVER['SYMFONY_DOTENV_VARS'] : '');
$output = array($listed);
foreach (explode(',', $listed) as $name) {
    if (isset($_SERVER[$name])) {
        $value = $_SERVER[$name];
    } elseif (isset($_ENV[$name])) {
        $value = $_ENV[$name];
    } else {
        continue;
    }
    $output[] = $name;
    $output[] = (string) $value;
}
echo implode("\0", $output);
