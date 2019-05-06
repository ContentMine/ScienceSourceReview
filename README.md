Webservice for reviewing articles stored on Science Source wikibase instance
=============================

This program is a simple server instance that lets people find and review articles stored in the Science Source wikibase instance.

Standing up ScienceSourceReview
-----------------------

It is expected that ScienceSourceReview will be stood up as a docker container as part of the standard ScienceSource wikibase instance. You will need to adjust the Dockerfile to copy the correct configuration file. There are two eaxmple files in the repo: `live.json` is values that should work with the production server, and `test.json` that has values that work in local testing.

BEFORE running Docker build you will need to create an OAuth Consumer Token/Secret pair and update the example configuration file. On the target wikibase server you should navigate to /wiki/Special:OAuthConsumerRegistration/propose and pick the following options:

* Application name - set to something you'll remember for this use
* Consumer version - doesn't matter, so set it to v1.0 or such
* Application description - set to something you'll remember

* Callback URL - Set this to the main URL for the wikibase site (e.g., http://sciencesource.wmflabs.org/ - include the http:// and final /)

The rest can remain at defaults. That last one is the most important, so please ensure you select that. If not then wikibase will require you to authorise the client via a web interface which will not work.

For Applicable grants select the following:

* Basic rights
* Edit existing pages
* Edit protected pages
* Create, edit, and move pages

When you click done you will find a page with the following strings on it:

* Consumer Token
* Consumer Secret

You should make a note of these and put them in the configuration JSON file.

Once that is done, check that your configuration JSON file name matches that in the Dockerfile in the repository, then build as normal.



License
============

This software is copyright Content Mine Ltd 2019, and released under the Apache 2.0 License.


Dependencies
============

Relies on

* https://github.com/ContentMine/wikibase
* https://github.com/mrjones/oauth
* https://github.com/gorilla/mux
* https://github.com/gorilla/sessions
* https://github.com/gorilla/securecookies
* https://github.com/flosch/pongo2
* https://github.com/juju/errors
