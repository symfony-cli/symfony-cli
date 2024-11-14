<?php


/*
 * This file is part of the Symfony package.
 *
 * (c) Fabien Potencier <fabien@symfony.com>
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

namespace Symfony\Requirements;

/**
 * Represents a single PHP requirement, e.g. an installed extension.
 * It can be a mandatory requirement or an optional recommendation.
 * There is a special subclass, named PhpConfigRequirement, to check a PHP
 * configuration option.
 *
 * @author Tobias Schultze <http://tobion.de>
 */
class Requirement
{
    private $fulfilled;
    private $testMessage;
    private $helpText;
    private $helpHtml;
    private $optional;

    /**
     * Constructor that initializes the requirement.
     *
     * @param bool        $fulfilled   Whether the requirement is fulfilled
     * @param string      $testMessage The message for testing the requirement
     * @param string      $helpHtml    The help text formatted in HTML for resolving the problem
     * @param string|null $helpText    The help text (when null, it will be inferred from $helpHtml, i.e. stripped from HTML tags)
     * @param bool        $optional    Whether this is only an optional recommendation not a mandatory requirement
     */
    public function __construct($fulfilled, $testMessage, $helpHtml, $helpText = null, $optional = false)
    {
        $this->fulfilled = (bool) $fulfilled;
        $this->testMessage = (string) $testMessage;
        $this->helpHtml = (string) $helpHtml;
        $this->helpText = null === $helpText ? strip_tags($this->helpHtml) : (string) $helpText;
        $this->optional = (bool) $optional;
    }

    /**
     * Returns whether the requirement is fulfilled.
     *
     * @return bool true if fulfilled, otherwise false
     */
    public function isFulfilled()
    {
        return $this->fulfilled;
    }

    /**
     * Returns the message for testing the requirement.
     *
     * @return string The test message
     */
    public function getTestMessage()
    {
        return $this->testMessage;
    }

    /**
     * Returns the help text for resolving the problem.
     *
     * @return string The help text
     */
    public function getHelpText()
    {
        return $this->helpText;
    }

    /**
     * Returns the help text formatted in HTML.
     *
     * @return string The HTML help
     */
    public function getHelpHtml()
    {
        return $this->helpHtml;
    }

    /**
     * Returns whether this is only an optional recommendation and not a mandatory requirement.
     *
     * @return bool true if optional, false if mandatory
     */
    public function isOptional()
    {
        return $this->optional;
    }
}


/*
 * This file is part of the Symfony package.
 *
 * (c) Fabien Potencier <fabien@symfony.com>
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

namespace Symfony\Requirements;

/**
 * Represents a requirement in form of a PHP configuration option.
 *
 * @author Tobias Schultze <http://tobion.de>
 */
class PhpConfigRequirement extends Requirement
{
    /**
     * Constructor that initializes the requirement.
     *
     * @param string        $cfgName           The configuration name used for ini_get()
     * @param bool|callable $evaluation        Either a boolean indicating whether the configuration should evaluate to true or false,
     *                                         or a callback function receiving the configuration value as parameter to determine the fulfillment of the requirement
     * @param bool          $approveCfgAbsence If true the Requirement will be fulfilled even if the configuration option does not exist, i.e. ini_get() returns false.
     *                                         This is helpful for abandoned configs in later PHP versions or configs of an optional extension, like Suhosin.
     *                                         Example: You require a config to be true but PHP later removes this config and defaults it to true internally.
     * @param string|null   $testMessage       The message for testing the requirement (when null and $evaluation is a boolean a default message is derived)
     * @param string|null   $helpHtml          The help text formatted in HTML for resolving the problem (when null and $evaluation is a boolean a default help is derived)
     * @param string|null   $helpText          The help text (when null, it will be inferred from $helpHtml, i.e. stripped from HTML tags)
     * @param bool          $optional          Whether this is only an optional recommendation not a mandatory requirement
     */
    public function __construct($cfgName, $evaluation, $approveCfgAbsence = false, $testMessage = null, $helpHtml = null, $helpText = null, $optional = false)
    {
        $cfgValue = ini_get($cfgName);

        if (is_callable($evaluation)) {
            if (null === $testMessage || null === $helpHtml) {
                throw new \InvalidArgumentException('You must provide the parameters testMessage and helpHtml for a callback evaluation.');
            }

            $fulfilled = call_user_func($evaluation, $cfgValue);
        } else {
            if (null === $testMessage) {
                $testMessage = sprintf('%s %s be %s in php.ini',
                    $cfgName,
                    $optional ? 'should' : 'must',
                    $evaluation ? 'enabled' : 'disabled'
                );
            }

            if (null === $helpHtml) {
                $helpHtml = sprintf('Set <strong>%s</strong> to <strong>%s</strong> in php.ini<a href="#phpini">*</a>.',
                    $cfgName,
                    $evaluation ? 'on' : 'off'
                );
            }

            $fulfilled = $evaluation == $cfgValue;
        }

        parent::__construct($fulfilled || ($approveCfgAbsence && false === $cfgValue), $testMessage, $helpHtml, $helpText, $optional);
    }
}


/*
 * This file is part of the Symfony package.
 *
 * (c) Fabien Potencier <fabien@symfony.com>
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

namespace Symfony\Requirements;

/**
 * A RequirementCollection represents a set of Requirement instances.
 *
 * @author Tobias Schultze <http://tobion.de>
 */
class RequirementCollection
{
    /**
     * @var Requirement[]
     */
    private $requirements = array();

    /**
     * Adds a Requirement.
     *
     * @param Requirement $requirement A Requirement instance
     */
    public function add(Requirement $requirement)
    {
        $this->requirements[] = $requirement;
    }

    /**
     * Adds a mandatory requirement.
     *
     * @param bool        $fulfilled   Whether the requirement is fulfilled
     * @param string      $testMessage The message for testing the requirement
     * @param string      $helpHtml    The help text formatted in HTML for resolving the problem
     * @param string|null $helpText    The help text (when null, it will be inferred from $helpHtml, i.e. stripped from HTML tags)
     */
    public function addRequirement($fulfilled, $testMessage, $helpHtml, $helpText = null)
    {
        $this->add(new Requirement($fulfilled, $testMessage, $helpHtml, $helpText, false));
    }

    /**
     * Adds an optional recommendation.
     *
     * @param bool        $fulfilled   Whether the recommendation is fulfilled
     * @param string      $testMessage The message for testing the recommendation
     * @param string      $helpHtml    The help text formatted in HTML for resolving the problem
     * @param string|null $helpText    The help text (when null, it will be inferred from $helpHtml, i.e. stripped from HTML tags)
     */
    public function addRecommendation($fulfilled, $testMessage, $helpHtml, $helpText = null)
    {
        $this->add(new Requirement($fulfilled, $testMessage, $helpHtml, $helpText, true));
    }

    /**
     * Adds a mandatory requirement in form of a PHP configuration option.
     *
     * @param string        $cfgName           The configuration name used for ini_get()
     * @param bool|callable $evaluation        Either a boolean indicating whether the configuration should evaluate to true or false,
     *                                         or a callback function receiving the configuration value as parameter to determine the fulfillment of the requirement
     * @param bool          $approveCfgAbsence If true the Requirement will be fulfilled even if the configuration option does not exist, i.e. ini_get() returns false.
     *                                         This is helpful for abandoned configs in later PHP versions or configs of an optional extension, like Suhosin.
     *                                         Example: You require a config to be true but PHP later removes this config and defaults it to true internally.
     * @param string        $testMessage       The message for testing the requirement (when null and $evaluation is a boolean a default message is derived)
     * @param string        $helpHtml          The help text formatted in HTML for resolving the problem (when null and $evaluation is a boolean a default help is derived)
     * @param string|null   $helpText          The help text (when null, it will be inferred from $helpHtml, i.e. stripped from HTML tags)
     */
    public function addPhpConfigRequirement($cfgName, $evaluation, $approveCfgAbsence = false, $testMessage = null, $helpHtml = null, $helpText = null)
    {
        $this->add(new PhpConfigRequirement($cfgName, $evaluation, $approveCfgAbsence, $testMessage, $helpHtml, $helpText, false));
    }

    /**
     * Adds an optional recommendation in form of a PHP configuration option.
     *
     * @param string        $cfgName           The configuration name used for ini_get()
     * @param bool|callable $evaluation        Either a boolean indicating whether the configuration should evaluate to true or false,
     *                                         or a callback function receiving the configuration value as parameter to determine the fulfillment of the requirement
     * @param bool          $approveCfgAbsence If true the Requirement will be fulfilled even if the configuration option does not exist, i.e. ini_get() returns false.
     *                                         This is helpful for abandoned configs in later PHP versions or configs of an optional extension, like Suhosin.
     *                                         Example: You require a config to be true but PHP later removes this config and defaults it to true internally.
     * @param string        $testMessage       The message for testing the requirement (when null and $evaluation is a boolean a default message is derived)
     * @param string        $helpHtml          The help text formatted in HTML for resolving the problem (when null and $evaluation is a boolean a default help is derived)
     * @param string|null   $helpText          The help text (when null, it will be inferred from $helpHtml, i.e. stripped from HTML tags)
     */
    public function addPhpConfigRecommendation($cfgName, $evaluation, $approveCfgAbsence = false, $testMessage = null, $helpHtml = null, $helpText = null)
    {
        $this->add(new PhpConfigRequirement($cfgName, $evaluation, $approveCfgAbsence, $testMessage, $helpHtml, $helpText, true));
    }

    /**
     * Adds a requirement collection to the current set of requirements.
     *
     * @param RequirementCollection $collection A RequirementCollection instance
     */
    public function addCollection(RequirementCollection $collection)
    {
        $this->requirements = array_merge($this->requirements, $collection->all());
    }

    /**
     * Returns both requirements and recommendations.
     *
     * @return Requirement[]
     */
    public function all()
    {
        return $this->requirements;
    }

    /**
     * Returns all mandatory requirements.
     *
     * @return Requirement[]
     */
    public function getRequirements()
    {
        $array = array();
        foreach ($this->requirements as $req) {
            if (!$req->isOptional()) {
                $array[] = $req;
            }
        }

        return $array;
    }

    /**
     * Returns the mandatory requirements that were not met.
     *
     * @return Requirement[]
     */
    public function getFailedRequirements()
    {
        $array = array();
        foreach ($this->requirements as $req) {
            if (!$req->isFulfilled() && !$req->isOptional()) {
                $array[] = $req;
            }
        }

        return $array;
    }

    /**
     * Returns all optional recommendations.
     *
     * @return Requirement[]
     */
    public function getRecommendations()
    {
        $array = array();
        foreach ($this->requirements as $req) {
            if ($req->isOptional()) {
                $array[] = $req;
            }
        }

        return $array;
    }

    /**
     * Returns the recommendations that were not met.
     *
     * @return Requirement[]
     */
    public function getFailedRecommendations()
    {
        $array = array();
        foreach ($this->requirements as $req) {
            if (!$req->isFulfilled() && $req->isOptional()) {
                $array[] = $req;
            }
        }

        return $array;
    }
}


/*
 * This file is part of the Symfony package.
 *
 * (c) Fabien Potencier <fabien@symfony.com>
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

namespace Symfony\Requirements;

/**
 * This class specifies all requirements and optional recommendations that
 * are necessary to run Symfony.
 *
 * @author Tobias Schultze <http://tobion.de>
 * @author Fabien Potencier <fabien@symfony.com>
 */
class ProjectRequirements extends RequirementCollection
{
    const REQUIRED_PHP_VERSION_3x = '5.5.9';
    const REQUIRED_PHP_VERSION_4x = '7.1.3';
    const REQUIRED_PHP_VERSION_5x = '7.2.9';
    const REQUIRED_PHP_VERSION_6x = '8.1.0';
    const REQUIRED_PHP_VERSION_7x = '8.2.0';

    public function __construct($rootDir)
    {
        $installedPhpVersion = phpversion();
        $symfonyVersion = null;
        if (file_exists($kernel = $rootDir.'/vendor/symfony/http-kernel/Kernel.php')) {
            $contents = file_get_contents($kernel);
            preg_match('{const VERSION += +\'([^\']+)\'}', $contents, $matches);
            $symfonyVersion = $matches[1];
        }

        $rootDir = $this->getComposerRootDir($rootDir);
        $options = $this->readComposer($rootDir);

        $phpVersion = self::REQUIRED_PHP_VERSION_7x;
        if (null !== $symfonyVersion) {
            if (version_compare($symfonyVersion, '6.0.0', '>=')) {
                $phpVersion = self::REQUIRED_PHP_VERSION_6x;
            } elseif (version_compare($symfonyVersion, '5.0.0', '>=')) {
                $phpVersion = self::REQUIRED_PHP_VERSION_5x;
            } elseif (version_compare($symfonyVersion, '4.0.0', '>=')) {
                $phpVersion = self::REQUIRED_PHP_VERSION_4x;
            } elseif (version_compare($symfonyVersion, '3.0.0', '>=')) {
                $phpVersion = self::REQUIRED_PHP_VERSION_3x;
            }
        }

        $this->addRequirement(
            version_compare($installedPhpVersion, $phpVersion, '>='),
            sprintf('PHP version must be at least %s (%s installed)', $phpVersion, $installedPhpVersion),
            sprintf('You are running PHP version "<strong>%s</strong>", but Symfony needs at least PHP "<strong>%s</strong>" to run.
            Before using Symfony, upgrade your PHP installation, preferably to the latest version.',
                $installedPhpVersion, $phpVersion),
            sprintf('Install PHP %s or newer (installed version is %s)', $phpVersion, $installedPhpVersion)
        );

        if (version_compare($installedPhpVersion, $phpVersion, '>=')) {
            $this->addRequirement(
                in_array(@date_default_timezone_get(), \DateTimeZone::listIdentifiers(), true),
                sprintf('Configured default timezone "%s" must be supported by your installation of PHP', @date_default_timezone_get()),
                'Your default timezone is not supported by PHP. Check for typos in your <strong>php.ini</strong> file and have a look at the list of deprecated timezones at <a href="http://php.net/manual/en/timezones.others.php">http://php.net/manual/en/timezones.others.php</a>.'
            );
        }

        $this->addRequirement(
            is_dir($rootDir.'/'.$options['vendor-dir'].'/composer'),
            'Vendor libraries must be installed',
            'Vendor libraries are missing. Install composer following instructions from <a href="http://getcomposer.org/">http://getcomposer.org/</a>. '.
            'Then run "<strong>php composer.phar install</strong>" to install them.'
        );

        if (is_dir($cacheDir = $rootDir.'/'.$options['var-dir'].'/cache')) {
            $this->addRequirement(
                is_writable($cacheDir),
                sprintf('%s/cache/ directory must be writable', $options['var-dir']),
                sprintf('Change the permissions of "<strong>%s/cache/</strong>" directory so that the web server can write into it.', $options['var-dir'])
            );
        }

        if (is_dir($logsDir = $rootDir.'/'.$options['var-dir'].'/log')) {
            $this->addRequirement(
                is_writable($logsDir),
                sprintf('%s/log/ directory must be writable', $options['var-dir']),
                sprintf('Change the permissions of "<strong>%s/log/</strong>" directory so that the web server can write into it.', $options['var-dir'])
            );
        }

        if (version_compare($installedPhpVersion, $phpVersion, '>=')) {
            $this->addRequirement(
                in_array(@date_default_timezone_get(), \DateTimeZone::listIdentifiers(), true),
                sprintf('Configured default timezone "%s" must be supported by your installation of PHP', @date_default_timezone_get()),
                'Your default timezone is not supported by PHP. Check for typos in your <strong>php.ini</strong> file and have a look at the list of deprecated timezones at <a href="http://php.net/manual/en/timezones.others.php">http://php.net/manual/en/timezones.others.php</a>.'
            );
        }
    }

    private function getComposerRootDir($rootDir)
    {
        $dir = $rootDir;
        while (!file_exists($dir.'/composer.json')) {
            if ($dir === dirname($dir)) {
                return $rootDir;
            }

            $dir = dirname($dir);
        }

        return $dir;
    }

    private function readComposer($rootDir)
    {
        $composer = json_decode(file_get_contents($rootDir.'/composer.json'), true);
        $options = array(
            'bin-dir' => 'bin',
            'conf-dir' => 'conf',
            'etc-dir' => 'etc',
            'src-dir' => 'src',
            'var-dir' => 'var',
            'public-dir' => 'public',
            'vendor-dir' => 'vendor',
        );

        foreach (array_keys($options) as $key) {
            if (isset($composer['extra'][$key])) {
                $options[$key] = $composer['extra'][$key];
            } elseif (isset($composer['extra']['symfony-'.$key])) {
                $options[$key] = $composer['extra']['symfony-'.$key];
            } elseif (isset($composer['config'][$key])) {
                $options[$key] = $composer['config'][$key];
            }
        }

        return $options;
    }
}


/*
 * This file is part of the Symfony package.
 *
 * (c) Fabien Potencier <fabien@symfony.com>
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

namespace Symfony\Requirements;

/**
 * This class specifies all requirements and optional recommendations that
 * are necessary to run Symfony.
 *
 * @author Tobias Schultze <http://tobion.de>
 * @author Fabien Potencier <fabien@symfony.com>
 */
class SymfonyRequirements extends RequirementCollection
{
    public function __construct()
    {
        $installedPhpVersion = phpversion();

        if (version_compare($installedPhpVersion, '7.0.0', '<')) {
            $this->addPhpConfigRequirement(
                'date.timezone', true, false,
                'date.timezone setting must be set',
                'Set the "<strong>date.timezone</strong>" setting in php.ini<a href="#phpini">*</a> (like Europe/Paris).'
            );
        }

        $this->addRequirement(
            function_exists('iconv'),
            'iconv() must be available',
            'Install and enable the <strong>iconv</strong> extension.'
        );

        $this->addRequirement(
            function_exists('json_encode'),
            'json_encode() must be available',
            'Install and enable the <strong>JSON</strong> extension.'
        );

        $this->addRequirement(
            function_exists('session_start'),
            'session_start() must be available',
            'Install and enable the <strong>session</strong> extension.'
        );

        $this->addRequirement(
            function_exists('ctype_alpha'),
            'ctype_alpha() must be available',
            'Install and enable the <strong>ctype</strong> extension.'
        );

        $this->addRequirement(
            function_exists('token_get_all'),
            'token_get_all() must be available',
            'Install and enable the <strong>Tokenizer</strong> extension.'
        );

        $this->addRequirement(
            function_exists('simplexml_import_dom'),
            'simplexml_import_dom() must be available',
            'Install and enable the <strong>SimpleXML</strong> extension.'
        );

        if (function_exists('apc_store') && ini_get('apc.enabled')) {
            if (version_compare($installedPhpVersion, '5.4.0', '>=')) {
                $this->addRequirement(
                    version_compare(phpversion('apc'), '3.1.13', '>='),
                    'APC version must be at least 3.1.13 when using PHP 5.4',
                    'Upgrade your <strong>APC</strong> extension (3.1.13+).'
                );
            } else {
                $this->addRequirement(
                    version_compare(phpversion('apc'), '3.0.17', '>='),
                    'APC version must be at least 3.0.17',
                    'Upgrade your <strong>APC</strong> extension (3.0.17+).'
                );
            }
        }

        $this->addPhpConfigRequirement('detect_unicode', false);

        if (extension_loaded('suhosin')) {
            $this->addPhpConfigRequirement(
                'suhosin.executor.include.whitelist',
                function($cfgValue) { return false !== stripos($cfgValue, 'phar'); },
                false,
                'suhosin.executor.include.whitelist must be configured correctly in php.ini',
                'Add "<strong>phar</strong>" to <strong>suhosin.executor.include.whitelist</strong> in php.ini<a href="#phpini">*</a>.'
            );
        }

        if (extension_loaded('xdebug')) {
            $this->addPhpConfigRequirement(
                'xdebug.show_exception_trace', false, true
            );

            $this->addPhpConfigRequirement(
                'xdebug.scream', false, true
            );

            $this->addPhpConfigRecommendation(
                'xdebug.max_nesting_level',
                function ($cfgValue) { return $cfgValue > 100; },
                true,
                'xdebug.max_nesting_level should be above 100 in php.ini',
                'Set "<strong>xdebug.max_nesting_level</strong>" to e.g. "<strong>250</strong>" in php.ini<a href="#phpini">*</a> to stop Xdebug\'s infinite recursion protection erroneously throwing a fatal error in your project.'
            );
        }

        $pcreVersion = defined('PCRE_VERSION') ? (float) PCRE_VERSION : null;

        $this->addRequirement(
            null !== $pcreVersion,
            'PCRE extension must be available',
            'Install the <strong>PCRE</strong> extension (version 8.0+).'
        );

        if (extension_loaded('mbstring')) {
            $this->addPhpConfigRequirement(
                'mbstring.func_overload',
                function ($cfgValue) { return (int) $cfgValue === 0; },
                true,
                'string functions should not be overloaded',
                'Set "<strong>mbstring.func_overload</strong>" to <strong>0</strong> in php.ini<a href="#phpini">*</a> to disable function overloading by the mbstring extension.'
            );
        }

        /* optional recommendations follow */

        if (null !== $pcreVersion) {
            $this->addRecommendation(
                $pcreVersion >= 8.0,
                sprintf('PCRE extension should be at least version 8.0 (%s installed)', $pcreVersion),
                '<strong>PCRE 8.0+</strong> is preconfigured in PHP since 5.3.2 but you are using an outdated version of it. Symfony probably works anyway but it is recommended to upgrade your PCRE extension.'
            );
        }

        $this->addRecommendation(
            class_exists('DomDocument'),
            'PHP-DOM and PHP-XML modules should be installed',
            'Install and enable the <strong>PHP-DOM</strong> and the <strong>PHP-XML</strong> modules.'
        );

        $this->addRecommendation(
            function_exists('mb_strlen'),
            'mb_strlen() should be available',
            'Install and enable the <strong>mbstring</strong> extension.'
        );

        $this->addRecommendation(
            function_exists('utf8_decode'),
            'utf8_decode() should be available',
            'Install and enable the <strong>XML</strong> extension.'
        );

        $this->addRecommendation(
            function_exists('filter_var'),
            'filter_var() should be available',
            'Install and enable the <strong>filter</strong> extension.'
        );

        if (!defined('PHP_WINDOWS_VERSION_BUILD')) {
            $this->addRecommendation(
                function_exists('posix_isatty'),
                'posix_isatty() should be available',
                'Install and enable the <strong>php_posix</strong> extension (used to colorize the CLI output).'
            );
        }

        $this->addRecommendation(
            extension_loaded('intl'),
            'intl extension should be available',
            'Install and enable the <strong>intl</strong> extension (used for validators).'
        );

        if (extension_loaded('intl')) {
            // in some WAMP server installations, new Collator() returns null
            $this->addRecommendation(
                null !== new \Collator('fr_FR'),
                'intl extension should be correctly configured',
                'The intl extension does not behave properly. This problem is typical on PHP 5.3.X x64 WIN builds.'
            );

            // check for compatible ICU versions (only done when you have the intl extension)
            if (defined('INTL_ICU_VERSION')) {
                $version = INTL_ICU_VERSION;
            } else {
                $reflector = new \ReflectionExtension('intl');

                ob_start();
                $reflector->info();
                $output = strip_tags(ob_get_clean());

                preg_match('/^ICU version +(?:=> )?(.*)$/m', $output, $matches);
                $version = $matches[1];
            }

            $this->addRecommendation(
                version_compare($version, '4.0', '>='),
                'intl ICU version should be at least 4+',
                'Upgrade your <strong>intl</strong> extension with a newer ICU version (4+).'
            );

            if (class_exists('Symfony\Component\Intl\Intl')) {
                $this->addRecommendation(
                    \Symfony\Component\Intl\Intl::getIcuDataVersion() <= \Symfony\Component\Intl\Intl::getIcuVersion(),
                    sprintf('intl ICU version installed on your system is outdated (%s) and does not match the ICU data bundled with Symfony (%s)', \Symfony\Component\Intl\Intl::getIcuVersion(), \Symfony\Component\Intl\Intl::getIcuDataVersion()),
                    'To get the latest internationalization data upgrade the ICU system package and the intl PHP extension.'
                );
                if (\Symfony\Component\Intl\Intl::getIcuDataVersion() <= \Symfony\Component\Intl\Intl::getIcuVersion()) {
                    $this->addRecommendation(
                        \Symfony\Component\Intl\Intl::getIcuDataVersion() === \Symfony\Component\Intl\Intl::getIcuVersion(),
                        sprintf('intl ICU version installed on your system (%s) does not match the ICU data bundled with Symfony (%s)', \Symfony\Component\Intl\Intl::getIcuVersion(), \Symfony\Component\Intl\Intl::getIcuDataVersion()),
                        'To avoid internationalization data inconsistencies upgrade the symfony/intl component.'
                    );
                }
            }

            $this->addPhpConfigRecommendation(
                'intl.error_level',
                function ($cfgValue) { return (int) $cfgValue === 0; },
                true,
                'intl.error_level should be 0 in php.ini',
                'Set "<strong>intl.error_level</strong>" to "<strong>0</strong>" in php.ini<a href="#phpini">*</a> to inhibit the messages when an error occurs in ICU functions.'
            );
        }

        $accelerator =
            (extension_loaded('eaccelerator') && ini_get('eaccelerator.enable'))
            ||
            (extension_loaded('apc') && ini_get('apc.enabled'))
            ||
            (extension_loaded('Zend Optimizer+') && ini_get('zend_optimizerplus.enable'))
            ||
            (extension_loaded('Zend OPcache') && ini_get('opcache.enable'))
            ||
            (extension_loaded('xcache') && ini_get('xcache.cacher'))
            ||
            (extension_loaded('wincache') && ini_get('wincache.ocenabled'))
        ;

        $this->addRecommendation(
            $accelerator,
            'a PHP accelerator should be installed',
            'Install and/or enable a <strong>PHP accelerator</strong> (highly recommended).'
        );

        if (strtoupper(substr(PHP_OS, 0, 3)) === 'WIN') {
            $this->addRecommendation(
                $this->getRealpathCacheSize() >= 5 * 1024 * 1024,
                'realpath_cache_size should be at least 5M in php.ini',
                'Setting "<strong>realpath_cache_size</strong>" to e.g. "<strong>5242880</strong>" or "<strong>5M</strong>" in php.ini<a href="#phpini">*</a> may improve performance on Windows significantly in some cases.'
            );
        }

        $this->addPhpConfigRecommendation('short_open_tag', false);

        $this->addPhpConfigRecommendation('magic_quotes_gpc', false, true);

        $this->addPhpConfigRecommendation('register_globals', false, true);

        $this->addPhpConfigRecommendation('session.auto_start', false);

        $this->addPhpConfigRecommendation(
            'xdebug.max_nesting_level',
            function ($cfgValue) { return $cfgValue > 100; },
            true,
            'xdebug.max_nesting_level should be above 100 in php.ini',
            'Set "<strong>xdebug.max_nesting_level</strong>" to e.g. "<strong>250</strong>" in php.ini<a href="#phpini">*</a> to stop Xdebug\'s infinite recursion protection erroneously throwing a fatal error in your project.'
        );

        $this->addPhpConfigRecommendation(
            'post_max_size',
            function () {
                $memoryLimit = $this->getMemoryLimit();
                $postMaxSize = $this->getPostMaxSize();

                return \INF === $memoryLimit || \INF === $postMaxSize || $memoryLimit > $postMaxSize;
            },
            true,
            '"memory_limit" should be greater than "post_max_size".',
            'Set "<strong>memory_limit</strong>" to be greater than "<strong>post_max_size</strong>".'
        );

        $this->addPhpConfigRecommendation(
            'upload_max_filesize',
            function () {
                $postMaxSize = $this->getPostMaxSize();
                $uploadMaxFilesize = $this->getUploadMaxFilesize();

                return \INF === $postMaxSize || \INF === $uploadMaxFilesize || $postMaxSize > $uploadMaxFilesize;
            },
            true,
            '"post_max_size" should be greater than "upload_max_filesize".',
            'Set "<strong>post_max_size</strong>" to be greater than "<strong>upload_max_filesize</strong>".'
        );

        $this->addRecommendation(
            class_exists('PDO'),
            'PDO should be installed',
            'Install <strong>PDO</strong> (mandatory for Doctrine).'
        );

        if (class_exists('PDO')) {
            $drivers = \PDO::getAvailableDrivers();
            $this->addRecommendation(
                count($drivers) > 0,
                sprintf('PDO should have some drivers installed (currently available: %s)', count($drivers) ? implode(', ', $drivers) : 'none'),
                'Install <strong>PDO drivers</strong> (mandatory for Doctrine).'
            );
        }
    }

    /**
     * Convert a given shorthand size in an integer
     * (e.g. 16k is converted to 16384 int)
     *
     * @param string $size Shorthand size
     * @param string $infiniteValue The infinite value for this setting
     *
     * @see http://www.php.net/manual/en/faq.using.php#faq.using.shorthandbytes
     *
     * @return float Converted size
     */
    private function convertShorthandSize($size, $infiniteValue = '-1')
    {
        // Initialize
        $size = trim($size);
        $unit = '';

        // Check unlimited alias
        if ($size === $infiniteValue) {
            return \INF;
        }

        // Check size
        if (!ctype_digit($size)) {
            $unit = strtolower(substr($size, -1, 1));
            $size = (int) substr($size, 0, -1);
        }

        // Return converted size
        switch ($unit) {
            case 'g':
                return $size * 1024 * 1024 * 1024;
            case 'm':
                return $size * 1024 * 1024;
            case 'k':
                return $size * 1024;
            default:
                return (int) $size;
        }
    }

    /**
     * Loads realpath_cache_size from php.ini and converts it to int.
     *
     * (e.g. 16k is converted to 16384 int)
     *
     * @return float
     */
    private function getRealpathCacheSize()
    {
        return $this->convertShorthandSize(ini_get('realpath_cache_size'));
    }

    /**
     * Loads post_max_size from php.ini and converts it to int.
     *
     * @return float
     */
    private function getPostMaxSize()
    {
        return $this->convertShorthandSize(ini_get('post_max_size'), '0');
    }

    /**
     * Loads memory_limit from php.ini and converts it to int.
     *
     * @return float
     */
    private function getMemoryLimit()
    {
        return $this->convertShorthandSize(ini_get('memory_limit'));
    }

    /**
     * Loads upload_max_filesize from php.ini and converts it to int.
     *
     * @return float
     */
    private function getUploadMaxFilesize()
    {
        return $this->convertShorthandSize(ini_get('upload_max_filesize'), '0');
    }
}


/*
 * This file is part of the Symfony package.
 *
 * (c) Fabien Potencier <fabien@symfony.com>
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

use Symfony\Requirements\Requirement;
use Symfony\Requirements\SymfonyRequirements;
use Symfony\Requirements\ProjectRequirements;

if (file_exists($autoloader = __DIR__.'/../../../autoload.php')) {
    require_once $autoloader;
} elseif (file_exists($autoloader = __DIR__.'/../vendor/autoload.php')) {
    require_once $autoloader;
} elseif (!class_exists('Symfony\Requirements\Requirement', false)) {
    require_once dirname(__DIR__).'/src/Requirement.php';
    require_once dirname(__DIR__).'/src/RequirementCollection.php';
    require_once dirname(__DIR__).'/src/PhpConfigRequirement.php';
    require_once dirname(__DIR__).'/src/SymfonyRequirements.php';
    require_once dirname(__DIR__).'/src/ProjectRequirements.php';
}

$lineSize = 70;
$args = array();
$isVerbose = false;
foreach ($argv as $arg) {
    if ('-v' === $arg || '-vv' === $arg || '-vvv' === $arg) {
        $isVerbose = true;
    } else {
        $args[] = $arg;
    }
}

$symfonyRequirements = new SymfonyRequirements();
$requirements = $symfonyRequirements->getRequirements();

// specific directory to check?
$dir = isset($args[1]) ? $args[1] : (file_exists(getcwd().'/composer.json') ? getcwd() : null);
if (null !== $dir) {
    $projectRequirements = new ProjectRequirements($dir);
    $requirements = array_merge($requirements, $projectRequirements->getRequirements());
}

echo_title('Symfony Requirements Checker');

echo '> PHP is using the following php.ini file:'.PHP_EOL;
if ($iniPath = get_cfg_var('cfg_file_path')) {
    echo_style('green', $iniPath);
} else {
    echo_style('yellow', 'WARNING: No configuration file (php.ini) used by PHP!');
}

echo PHP_EOL.PHP_EOL;

echo '> Checking Symfony requirements:'.PHP_EOL.PHP_EOL;

$messages = array();
foreach ($requirements as $req) {
    if ($helpText = get_error_message($req, $lineSize)) {
        if ($isVerbose) {
            echo_style('red', '[ERROR] ');
            echo $req->getTestMessage().PHP_EOL;
        } else {
            echo_style('red', 'E');
        }

        $messages['error'][] = $helpText;
    } else {
        if ($isVerbose) {
            echo_style('green', '[OK] ');
            echo $req->getTestMessage().PHP_EOL;
        } else {
            echo_style('green', '.');
        }
    }
}

$checkPassed = empty($messages['error']);

foreach ($symfonyRequirements->getRecommendations() as $req) {
    if ($helpText = get_error_message($req, $lineSize)) {
        if ($isVerbose) {
            echo_style('yellow', '[WARN] ');
            echo $req->getTestMessage().PHP_EOL;
        } else {
            echo_style('yellow', 'W');
        }

        $messages['warning'][] = $helpText;
    } else {
        if ($isVerbose) {
            echo_style('green', '[OK] ');
            echo $req->getTestMessage().PHP_EOL;
        } else {
            echo_style('green', '.');
        }
    }
}

if ($checkPassed) {
    echo_block('success', 'OK', 'Your system is ready to run Symfony projects');
} else {
    echo_block('error', 'ERROR', 'Your system is not ready to run Symfony projects');

    echo_title('Fix the following mandatory requirements', 'red');

    foreach ($messages['error'] as $helpText) {
        echo ' * '.$helpText.PHP_EOL;
    }
}

if (!empty($messages['warning'])) {
    echo_title('Optional recommendations to improve your setup', 'yellow');

    foreach ($messages['warning'] as $helpText) {
        echo ' * '.$helpText.PHP_EOL;
    }
}

echo PHP_EOL;
echo_style('title', 'Note');
echo '  The command console can use a different php.ini file'.PHP_EOL;
echo_style('title', '~~~~');
echo '  than the one used by your web server.'.PHP_EOL;
echo '      Please check that both the console and the web server'.PHP_EOL;
echo '      are using the same PHP version and configuration.'.PHP_EOL;
echo PHP_EOL;

exit($checkPassed ? 0 : 1);

function get_error_message(Requirement $requirement, $lineSize)
{
    if ($requirement->isFulfilled()) {
        return;
    }

    $errorMessage = wordwrap($requirement->getTestMessage(), $lineSize - 3, PHP_EOL.'   ').PHP_EOL;
    $errorMessage .= '   > '.wordwrap($requirement->getHelpText(), $lineSize - 5, PHP_EOL.'   > ').PHP_EOL;

    return $errorMessage;
}

function echo_title($title, $style = null)
{
    $style = $style ?: 'title';

    echo PHP_EOL;
    echo_style($style, $title.PHP_EOL);
    echo_style($style, str_repeat('~', strlen($title)).PHP_EOL);
    echo PHP_EOL;
}

function echo_style($style, $message)
{
    // ANSI color codes
    $styles = array(
        'reset' => "\033[0m",
        'red' => "\033[31m",
        'green' => "\033[32m",
        'yellow' => "\033[33m",
        'error' => "\033[37;41m",
        'success' => "\033[37;42m",
        'title' => "\033[34m",
    );
    $supports = has_color_support();

    echo($supports ? $styles[$style] : '').$message.($supports ? $styles['reset'] : '');
}

function echo_block($style, $title, $message)
{
    $message = ' '.trim($message).' ';
    $width = strlen($message);

    echo PHP_EOL.PHP_EOL;

    echo_style($style, str_repeat(' ', $width));
    echo PHP_EOL;
    echo_style($style, str_pad(' ['.$title.']', $width, ' ', STR_PAD_RIGHT));
    echo PHP_EOL;
    echo_style($style, $message);
    echo PHP_EOL;
    echo_style($style, str_repeat(' ', $width));
    echo PHP_EOL;
}

function has_color_support()
{
    static $support;

    if (null === $support) {

        if ('Hyper' === getenv('TERM_PROGRAM')) {
            return $support = true;
        }

        if (DIRECTORY_SEPARATOR === '\\') {
            return  $support = (function_exists('sapi_windows_vt100_support')
                && @sapi_windows_vt100_support(STDOUT))
                || false !== getenv('ANSICON')
                || 'ON' === getenv('ConEmuANSI')
                || 'xterm' === getenv('TERM');
        }

        if (function_exists('stream_isatty')) {
            return $support = @stream_isatty(STDOUT);
        }

        if (function_exists('posix_isatty')) {
            return $support = @posix_isatty(STDOUT);
        }

        $stat = @fstat(STDOUT);
        // Check if formatted mode is S_IFCHR
        return $support = ( $stat ? 0020000 === ($stat['mode'] & 0170000) : false );
    }

    return $support;
}
