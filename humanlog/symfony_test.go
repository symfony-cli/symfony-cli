/*
 * Copyright (c) 2021-present Fabien Potencier <fabien@symfony.com>
 *
 * This file is part of Symfony CLI project
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <http://www.gnu.org/licenses/>.
 */

package humanlog

// [awards] [2017-10-03 21:38:50] request.ERROR: Uncaught PHP Exception Symfony\Component\HttpKernel\Exception\NotFoundHttpException: "No route found for "GET /community"" at /app/vendor/symfony/http-kernel/EventListener/RouterListener.php line 125 {"exception":"[object] (Symfony\\Component\\HttpKernel\\Exception\\NotFoundHttpException(code: 0): No route found for \"GET /community\" at /app/vendor/symfony/http-kernel/EventListener/RouterListener.php:125, Symfony\\Component\\Routing\\Exception\\ResourceNotFoundException(code: 0):  at /app/var/cache/prod/srcProdProjectContainerUrlMatcher.php:104)"} []
// [awards] [2017-10-04 00:27:08] request.INFO: Matched route "homepage". {"route":"homepage","route_parameters":{"_controller":"App\\Controller\\ContentController::homeAction","_route":"homepage"},"request_uri":"https://awards.symfony.com/","method":"GET"} []
// [awards] [2017-10-03 21:30:09] request.CRITICAL: Uncaught PHP Exception Twig_Error_Loader: "Unable to find template "Connect/script.html.twig" (looked into: /app/templates, /app/vendor/symfony/twig-bridge/Resources/views/Form)." at /app/templates/layout.html.twig line 31 {"exception":"[object] (Twig_Error_Loader(code: 0): Unable to find template \"Connect/script.html.twig\" (looked into: /app/templates, /app/vendor/symfony/twig-bridge/Resources/views/Form). at /app/templates/layout.html.twig:31)"} []

import (
	"testing"
	"time"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type HumanlogSuite struct{}

var _ = Suite(&HumanlogSuite{})

func (s *HumanlogSuite) TestSymfonyLogConverter(c *C) {
	now := time.Now()
	ins := []string{
		`[2018-01-29 07:08:59] security.DEBUG: User was reloaded from a user provider. {"username":"tucksaun","provider":"Symfony\\Bridge\\Doctrine\\Security\\User\\EntityUserProvider"} {"log_uuid":"32e5445b-379c-4f7b-a81f-23466ef1dd59","url":"/admin/dashboard","ip":"127.0.0.1","http_method":"GET","server":"127.0.0.1","referrer":"http://127.0.0.1:8000/cloud/"}`,
		`[2017-10-04 00:27:09] security.INFO: Populated the TokenStorage with an anonymous Token. [] []`,
		`[2018-01-29 07:08:59] doctrine.DEBUG: SELECT t0.id [1] {"log_uuid":"32e5445b-379c-4f7b-a81f-23466ef1dd59","url":"/admin/dashboard","ip":"127.0.0.1","http_method":"GET","server":"127.0.0.1","referrer":"http://127.0.0.1:8000/cloud/"}`,
		`[2018-01-29 07:08:59] app.INFO: [cms::renderBlock] block.id=5, block.type=sonata.admin.block.admin_list [] {"user":{"uuid":"01888dba-6735-4013-b1dc-e16e94bea611"},"log_uuid":"32e5445b-379c-4f7b-a81f-23466ef1dd59","url":"/admin/dashboard","ip":"127.0.0.1","http_method":"GET","server":"127.0.0.1","referrer":"http://127.0.0.1:8000/cloud/"}`,
		`[2018-12-04 17:09:48] request.CRITICAL: Uncaught PHP Exception Twig_Error_Runtime: "An exception has been thrown during the rendering of a template ("An exception occurred while executing 'SELECT COUNT(*) AS dctrn_count FROM (SELECT DISTINCT id_0 FROM (SELECT s0_.id AS id_0, s0_.title AS title_1, s0_.slug AS slug_2, s0_.summary AS summary_3, s0_.content AS content_4, s0_.published_at AS published_at_5, s1_.id AS id_6, s1_.full_name AS full_name_7, s1_.username AS username_8, s1_.email AS email_9, s1_.password AS password_10, s1_.roles AS roles_11, s2_.id AS id_12, s2_.name AS name_13 FROM symfony_demo_post s0_ INNER JOIN symfony_demo_user s1_ ON s0_.author_id = s1_.id LEFT JOIN symfony_demo_post_tag s3_ ON s0_.id = s3_.post_id LEFT JOIN symfony_demo_tag s2_ ON s2_.id = s3_.tag_id WHERE s0_.published_at <= ? ORDER BY s0_.published_at DESC, s2_.name ASC) dctrn_result) dctrn_table' with params ["2018-12-04 17:09:48"]:  SQLSTATE[HY000]: General error: 1 no such table: symfony_demo_post")." at /app/templates/blog/index.html.twig line 6 {"exception":"[object] (Twig_Error_Runtime(code: 0): An exception has been thrown during the rendering of a template (\"An exception occurred while executing 'SELECT COUNT(*) AS dctrn_count FROM (SELECT DISTINCT id_0 FROM (SELECT s0_.id AS id_0, s0_.title AS title_1, s0_.slug AS slug_2, s0_.summary AS summary_3, s0_.content AS content_4, s0_.published_at AS published_at_5, s1_.id AS id_6, s1_.full_name AS full_name_7, s1_.username AS username_8, s1_.email AS email_9, s1_.password AS password_10, s1_.roles AS roles_11, s2_.id AS id_12, s2_.name AS name_13 FROM symfony_demo_post s0_ INNER JOIN symfony_demo_user s1_ ON s0_.author_id = s1_.id LEFT JOIN symfony_demo_post_tag s3_ ON s0_.id = s3_.post_id LEFT JOIN symfony_demo_tag s2_ ON s2_.id = s3_.tag_id WHERE s0_.published_at <= ? ORDER BY s0_.published_at DESC, s2_.name ASC) dctrn_result) dctrn_table' with params [\"2018-12-04 17:09:48\"]:\n\nSQLSTATE[HY000]: General error: 1 no such table: symfony_demo_post\"). at /app/templates/blog/index.html.twig:6, Doctrine\\DBAL\\Exception\\TableNotFoundException(code: 0): An exception occurred while executing 'SELECT COUNT(*) AS dctrn_count FROM (SELECT DISTINCT id_0 FROM (SELECT s0_.id AS id_0, s0_.title AS title_1, s0_.slug AS slug_2, s0_.summary AS summary_3, s0_.content AS content_4, s0_.published_at AS published_at_5, s1_.id AS id_6, s1_.full_name AS full_name_7, s1_.username AS username_8, s1_.email AS email_9, s1_.password AS password_10, s1_.roles AS roles_11, s2_.id AS id_12, s2_.name AS name_13 FROM symfony_demo_post s0_ INNER JOIN symfony_demo_user s1_ ON s0_.author_id = s1_.id LEFT JOIN symfony_demo_post_tag s3_ ON s0_.id = s3_.post_id LEFT JOIN symfony_demo_tag s2_ ON s2_.id = s3_.tag_id WHERE s0_.published_at <= ? ORDER BY s0_.published_at DESC, s2_.name ASC) dctrn_result) dctrn_table' with params [\"2018-12-04 17:09:48\"]:\n\nSQLSTATE[HY000]: General error: 1 no such table: symfony_demo_post at /app/vendor/doctrine/dbal/lib/Doctrine/DBAL/Driver/AbstractSQLiteDriver.php:63, Doctrine\\DBAL\\Driver\\PDOException(code: HY000): SQLSTATE[HY000]: General error: 1 no such table: symfony_demo_post at /app/vendor/doctrine/dbal/lib/Doctrine/DBAL/Driver/PDOConnection.php:82, PDOException(code: HY000): SQLSTATE[HY000]: General error: 1 no such table: symfony_demo_post at /app/vendor/doctrine/dbal/lib/Doctrine/DBAL/Driver/PDOConnection.php:80)"} []`,
		`[2019-11-13T07:22:27.870797+01:00] security.DEBUG: Checking support on guard authenticator. {"firewall_key":"main","authenticator":"App\\Security\\AppAuthenticator"} []`,
		`[2019-11-13 07:10:44] request.CRITICAL: Exception thrown when handling an exception (LogicException: An instance of Symfony\Component\Templating\EngineInterface must be injected in FOS\RestBundle\View\ViewHandler to render templates. at /home/fabien/Code/github/blackfireio/blackfire.io/app.blackfire.io/vendor/friendsofsymfony/rest-bundle/View/ViewHandler.php line 366) {"exception":{"type":"LogicException","message":"An instance of Symfony\\Component\\Templating\\EngineInterface must be injected in FOS\\RestBundle\\View\\ViewHandler to render templates.","code":0,"file":"/home/fabien/Code/github/blackfireio/blackfire.io/app.blackfire.io/vendor/friendsofsymfony/rest-bundle/View/ViewHandler.php","trace":"#0 /home/fabien/Code/github/blackfireio/blackfire.io/app.blackfire.io/vendor/friendsofsymfony/rest-bundle/View/ViewHandler.php(462): FOS\\RestBundle\\View\\ViewHandler->renderTemplate(Object(FOS\\RestBundle\\View\\View), 'html')\n#1 /home/fabien/Code/github/blackfireio/blackfire.io/app.blackfire.io/vendor/friendsofsymfony/rest-bundle/View/ViewHandler.php(436): FOS\\RestBundle\\View\\ViewHandler->initResponse(Object(FOS\\RestBundle\\View\\View), 'html')\n#2 /home/fabien/Code/github/blackfireio/blackfire.io/app.blackfire.io/vendor/friendsofsymfony/rest-bundle/View/ViewHandler.php(320): FOS\\RestBundle\\View\\ViewHandler->createResponse(Object(FOS\\RestBundle\\View\\View), Object(Symfony\\Component\\HttpFoundation\\Request), 'html')\n#3 /home/fabien/Code/github/blackfireio/blackfire.io/app.blackfire.io/vendor/friendsofsymfony/rest-bundle/Controller/ExceptionController.php(72): FOS\\RestBundle\\View\\ViewHandler->handle(Object(FOS\\RestBundle\\View\\View))\n#4 /home/fabien/Code/github/blackfireio/blackfire.io/app.blackfire.io/vendor/symfony/http-kernel/HttpKernel.php(151): FOS\\RestBundle\\Controller\\ExceptionController->showAction(Object(Symfony\\Component\\HttpFoundation\\Request), Object(Doctrine\\DBAL\\Exception\\ConnectionException), Object(SensioLabs\\Toolkit\\Monolog\\Logger))\n#5 /home/fabien/Code/github/blackfireio/blackfire.io/app.blackfire.io/vendor/symfony/http-kernel/HttpKernel.php(68): Symfony\\Component\\HttpKernel\\HttpKernel->handleRaw(Object(Symfony\\Component\\HttpFoundation\\Request), 2)\n#6 /home/fabien/Code/github/blackfireio/blackfire.io/app.blackfire.io/vendor/symfony/http-kernel/EventListener/ExceptionListener.php(62): Symfony\\Component\\HttpKernel\\HttpKernel->handle(Object(Symfony\\Component\\HttpFoundation\\Request), 2, false)\n#7 /home/fabien/Code/github/blackfireio/blackfire.io/app.blackfire.io/vendor/friendsofsymfony/rest-bundle/EventListener/ExceptionListener.php(41): Symfony\\Component\\HttpKernel\\EventListener\\ExceptionListener->onKernelException(Object(Symfony\\Component\\HttpKernel\\Event\\ExceptionEvent))\n#8 /home/fabien/Code/github/blackfireio/blackfire.io/app.blackfire.io/vendor/symfony/event-dispatcher/Debug/WrappedListener.php(126): FOS\\RestBundle\\EventListener\\ExceptionListener->onKernelException(Object(Symfony\\Component\\HttpKernel\\Event\\ExceptionEvent), 'kernel.exceptio...', Object(Symfony\\Component\\HttpKernel\\Debug\\TraceableEventDispatcher))\n#9 /home/fabien/Code/github/blackfireio/blackfire.io/app.blackfire.io/vendor/symfony/event-dispatcher/EventDispatcher.php(260): Symfony\\Component\\EventDispatcher\\Debug\\WrappedListener->__invoke(Object(Symfony\\Component\\HttpKernel\\Event\\ExceptionEvent), 'kernel.exceptio...', Object(Symfony\\Component\\HttpKernel\\Debug\\TraceableEventDispatcher))\n#10 /home/fabien/Code/github/blackfireio/blackfire.io/app.blackfire.io/vendor/symfony/event-dispatcher/EventDispatcher.php(235): Symfony\\Component\\EventDispatcher\\EventDispatcher->doDispatch(Array, 'kernel.exceptio...', Object(Symfony\\Component\\HttpKernel\\Event\\ExceptionEvent))\n#11 /home/fabien/Code/github/blackfireio/blackfire.io/app.blackfire.io/vendor/symfony/event-dispatcher/EventDispatcher.php(73): Symfony\\Component\\EventDispatcher\\EventDispatcher->callListeners(Array, 'kernel.exceptio...', Object(Symfony\\Component\\HttpKernel\\Event\\ExceptionEvent))\n#12 /home/fabien/Code/github/blackfireio/blackfire.io/app.blackfire.io/vendor/symfony/event-dispatcher/Debug/TraceableEventDispatcher.php(168): Symfony\\Component\\EventDispatcher\\EventDispatcher->dispatch(Object(Symfony\\Component\\HttpKernel\\Event\\ExceptionEvent), 'kernel.exceptio...')\n#13 /home/fabien/Code/github/blackfireio/blackfire.io/app.blackfire.io/vendor/symfony/http-kernel/HttpKernel.php(222): Symfony\\Component\\EventDispatcher\\Debug\\TraceableEventDispatcher->dispatch(Object(Symfony\\Component\\HttpKernel\\Event\\ExceptionEvent), 'kernel.exceptio...')\n#14 /home/fabien/Code/github/blackfireio/blackfire.io/app.blackfire.io/vendor/symfony/http-kernel/HttpKernel.php(79): Symfony\\Component\\HttpKernel\\HttpKernel->handleException(Object(Doctrine\\DBAL\\Exception\\ConnectionException), Object(Symfony\\Component\\HttpFoundation\\Request), 1)\n#15 /home/fabien/Code/github/blackfireio/blackfire.io/app.blackfire.io/vendor/symfony/http-kernel/Kernel.php(198): Symfony\\Component\\HttpKernel\\HttpKernel->handle(Object(Symfony\\Component\\HttpFoundation\\Request), 1, true)\n#16 /home/fabien/Code/github/blackfireio/blackfire.io/app.blackfire.io/web/app.php(26): Symfony\\Component\\HttpKernel\\Kernel->handle(Object(Symfony\\Component\\HttpFoundation\\Request))\n#17 /home/fabien/.symfony/php/7c4624a13b7623898d189d18788082aa8f80d6bb-router.php(26): require('/home/fabien/Co...')\n#18 {main}"}} {"log_uuid":"13850f2a-9419-4d28-b66a-8509ecb697b4","SAPI":"cli-server","url":"/","ip":"127.0.0.1","http_method":"GET","server":"blackfire.wip","referrer":null}`,
	}
	expected := []*line{
		{
			time:    now,
			level:   "debug",
			source:  "security",
			message: "User was reloaded from a user provider.",
			fields: map[string]string{
				"username":    "\"tucksaun\"",
				"provider":    "\"Symfony\\\\Bridge\\\\Doctrine\\\\Security\\\\User\\\\EntityUserProvider\"",
				"log_uuid":    "\"32e5445b-379c-4f7b-a81f-23466ef1dd59\"",
				"url":         "\"/admin/dashboard\"",
				"ip":          "\"127.0.0.1\"",
				"http_method": "\"GET\"",
				"server":      "\"127.0.0.1\"",
				"referrer":    "\"http://127.0.0.1:8000/cloud/\"",
			},
		},
		{
			time:    now,
			level:   "info",
			source:  "security",
			message: "Populated the TokenStorage with an anonymous Token.",
			fields:  map[string]string{},
		},
		{
			time:    now,
			level:   "debug",
			source:  "doctrine",
			message: "SELECT t0.id",
			fields: map[string]string{
				"0":           "1",
				"log_uuid":    "\"32e5445b-379c-4f7b-a81f-23466ef1dd59\"",
				"url":         "\"/admin/dashboard\"",
				"ip":          "\"127.0.0.1\"",
				"http_method": "\"GET\"",
				"server":      "\"127.0.0.1\"",
				"referrer":    "\"http://127.0.0.1:8000/cloud/\"",
			},
		},
		{
			time:    now,
			level:   "info",
			source:  "app",
			message: "[cms::renderBlock] block.id=5, block.type=sonata.admin.block.admin_list",
			fields: map[string]string{
				"user":        "{\"uuid\":\"01888dba-6735-4013-b1dc-e16e94bea611\"}",
				"log_uuid":    "\"32e5445b-379c-4f7b-a81f-23466ef1dd59\"",
				"url":         "\"/admin/dashboard\"",
				"ip":          "\"127.0.0.1\"",
				"http_method": "\"GET\"",
				"server":      "\"127.0.0.1\"",
				"referrer":    "\"http://127.0.0.1:8000/cloud/\"",
			},
		},
		{
			time:    now,
			level:   "critical",
			source:  "request",
			message: `Uncaught PHP Exception Twig_Error_Runtime: "An exception has been thrown during the rendering of a template ("An exception occurred while executing 'SELECT COUNT(*) AS dctrn_count FROM (SELECT DISTINCT id_0 FROM (SELECT s0_.id AS id_0, s0_.title AS title_1, s0_.slug AS slug_2, s0_.summary AS summary_3, s0_.content AS content_4, s0_.published_at AS published_at_5, s1_.id AS id_6, s1_.full_name AS full_name_7, s1_.username AS username_8, s1_.email AS email_9, s1_.password AS password_10, s1_.roles AS roles_11, s2_.id AS id_12, s2_.name AS name_13 FROM symfony_demo_post s0_ INNER JOIN symfony_demo_user s1_ ON s0_.author_id = s1_.id LEFT JOIN symfony_demo_post_tag s3_ ON s0_.id = s3_.post_id LEFT JOIN symfony_demo_tag s2_ ON s2_.id = s3_.tag_id WHERE s0_.published_at <= ? ORDER BY s0_.published_at DESC, s2_.name ASC) dctrn_result) dctrn_table' with params ["2018-12-04 17:09:48"]:  SQLSTATE[HY000]: General error: 1 no such table: symfony_demo_post")." at /app/templates/blog/index.html.twig line 6`,
			fields:  map[string]string{},
		},
		{
			time:    now,
			level:   "debug",
			source:  "security",
			message: `Checking support on guard authenticator.`,
			fields: map[string]string{
				"firewall_key":  `"main"`,
				"authenticator": `"App\\Security\\AppAuthenticator"`,
			},
		},
		{
			time:    now,
			level:   "critical",
			source:  "request",
			message: `Exception thrown when handling an exception (LogicException: An instance of Symfony\Component\Templating\EngineInterface must be injected in FOS\RestBundle\View\ViewHandler to render templates. at /home/fabien/Code/github/blackfireio/blackfire.io/app.blackfire.io/vendor/friendsofsymfony/rest-bundle/View/ViewHandler.php line 366)`,
			fields: map[string]string{
				"log_uuid":    "\"13850f2a-9419-4d28-b66a-8509ecb697b4\"",
				"url":         "\"/\"",
				"ip":          "\"127.0.0.1\"",
				"http_method": "\"GET\"",
				"server":      "\"blackfire.wip\"",
				"SAPI":        "\"cli-server\"",
				"referrer":    "null",
			},
		},
	}
	for i, in := range ins {
		out, err := convertSymfonyLog([]byte(in))
		c.Assert(err, Equals, nil)
		if out != nil {
			out.time = now
		}
		c.Assert(out, DeepEquals, expected[i])
	}
}
