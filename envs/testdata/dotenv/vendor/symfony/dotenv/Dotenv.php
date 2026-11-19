<?php

namespace Symfony\Component\Dotenv;

// Stand-in for symfony/dotenv with the same file cascade and precedence rules,
// but only parsing NAME=value lines.
final class Dotenv
{
    public function bootEnv(string $path): void
    {
        $this->loadEnv($path);
        $_SERVER['APP_DEBUG'] = $_ENV['APP_DEBUG'] = '1';
    }

    public function loadEnv(string $path): void
    {
        $this->load(is_file($path) || !is_file("$path.dist") ? $path : "$path.dist");
        if (null === $env = $_SERVER['APP_ENV'] ?? $_ENV['APP_ENV'] ?? null) {
            $this->populate(['APP_ENV' => $env = 'dev']);
        }
        if ('test' !== $env && is_file("$path.local")) {
            $this->load("$path.local");
            $env = $_SERVER['APP_ENV'] ?? $_ENV['APP_ENV'] ?? $env;
        }
        if ('local' === $env) {
            return;
        }
        foreach (["$path.$env", "$path.$env.local"] as $p) {
            if (is_file($p)) {
                $this->load($p);
            }
        }
    }

    public function load(string $path): void
    {
        $values = [];
        foreach (file($path, \FILE_IGNORE_NEW_LINES | \FILE_SKIP_EMPTY_LINES) as $line) {
            if (!str_contains($line, '=')) {
                throw new \RuntimeException(\sprintf('Missing = in "%s".', $path));
            }
            [$name, $value] = explode('=', $line, 2);
            $values[$name] = $value;
        }
        $this->populate($values);
    }

    public function populate(array $values): void
    {
        $loadedVars = array_flip(explode(',', $_SERVER['SYMFONY_DOTENV_VARS'] ?? $_ENV['SYMFONY_DOTENV_VARS'] ?? ''));
        unset($loadedVars['']);
        foreach ($values as $name => $value) {
            if (isset($_SERVER[$name]) && !isset($_ENV[$name])) {
                $_ENV[$name] = $_SERVER[$name];
            }
            if (!isset($loadedVars[$name]) && isset($_ENV[$name])) {
                continue;
            }
            $_ENV[$name] = $_SERVER[$name] = $value;
            $loadedVars[$name] = true;
        }
        $_ENV['SYMFONY_DOTENV_VARS'] = $_SERVER['SYMFONY_DOTENV_VARS'] = implode(',', array_keys($loadedVars));
    }
}
