# feedproxy

## Overview

Feedproxy is a simple and lightweigt proxy to host modified versions of rss feeds.
Some feeds only provide links to articles, some pages have no feed at all and some do strange stuff (looking at you, "Cached by Wordpress plugin xy" message which breaks the parser of my reader).

This proxy can be hosed aside with a selfhosted feed reader (TinyTinyRSS, Miniflux, etc.) to fix feeds which no not fullfill the own expectations.
The feeds are accessible from any subfolder, thereby an arbitrary unused path of another host (e.g. a selfhosted feedreader) can be routed to the proxy to host the modified feeds.

## Implemented feeds

### Comic strips
* http://dilbert.com actual comic images are inserted into the feed items
* http://www.thegamercat.com tiny preview images are replaced with full size comic
* https://ruthe.de build missing feed from scratch
* https://www.commitstrip.com remove wordpress cache info at end of document
* https://joscha.com/nichtlustig build missing feed from scratch

### https://www.heise.de
As some news items appear on multiple feeds, filtered versions for heiseOnline, heiseSecurity, heiseDeveloper and iX are created to make each news item only appear on one of the feeds.
    
## Open Tasks
* Caching - some feeds query every article on every request. Some kind of cache would be useful to remove load from the original servers
