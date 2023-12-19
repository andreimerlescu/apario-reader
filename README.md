# Apario Reader

This project is part of the Project Apario open source offering that will provide  a web
graphical user interface (GUI) for the 
[Apario Contribution](https://github.com/andreimerlescu/apario-contribution) project.


## Dependencies 

The contribution project is extremely resource intensive as it builds the database for 
this project. When you run the contribution project, the output `tmp` directory will 
become the `database/<collection>`. For example, if you compile the results for the 
*STARGATE* collection, you'll generate around 170GB of data and on a 16-core 3.6GHz 
server with 64GB of RAM, it'll take you 1 day 17 hours 33 minutes to complete (36pgs/min).
With that generated output, you'd place that output `tmp` directory in the 
`database/stargate` directory of this project (symbolic links are supported). In addition
to the compiled database that is required, you'll also need to acquire (elsewhere) the 
`bundled/geography/locations.csv` file yourself. 

I purchased 
[SimpleMaps World Cities](https://simplemaps.com/data/world-cities) for $499 for 
Project Apario in 2020. However, we weren't able to properly use it in the application
because of technical limitations in the implementation of the Ruby on Rails version of the
site. However, the application was built on top of that offering, so if you want to run 
this service yourself using the full open source copy of the Golang rewrite of the app, 
you can purchase it yourself and leverage the GUI functionality that GPS coordinates 
offer to full text analysis extraction. Per the 
[license](https://simplemaps.com/data/license), I cannot redistribute the 4.4M rows CSV
file.

## Runtime

During the development of this project, the *STARGATE* collection from the
[Apario Contribution](https://github.com/andreimerlescu/apario-contribution) project was 
used and when the application  boots it takes about 5 minutes to load and consumes around 
12GB of RAM once running. This value is directly proportional to the size of the 
`database/<collection>` used. For instance, the JFK 2018 release is over 2TB of data,
whereas the STARGATE files are 170GB of data. This means that when the JFK 2018 data
set is loaded, it'll consume much more data. Just using basic math, 12GB / 90K pages 
comes out to 133KB per page of data being stored in memory. This makes sense since the
full text of the page is in memory PLUS the metadata indexes are in memory. Therefore
when expanding to 334K pages from 90K pages, the memory usage will be ~133KB * 334K
which comes out to 44.4GB of RAM. For many servers, this is fine and is not too much
to deal with, but if you're trying to run this application on a MacBook Air, don't.
There is and was a tradeoff on this decision based on the runtime of application. 
Most services are not built this way and most services will use microservices to
divide the massive computational resources into various components. However, to put
things into perspective, when Project Apario was operational from 2020 to 2023 it
was running on 6-12 physical bare metal servers that all had 64GB of RAM on each
of them and the MongoDB database specifically was running in a cluster of 3 bare
metal hosts that each had 256GB of RAM on them. This represents 384GB of RAM for
the web application and 768GB of RAM for the database. Why so much? Advanced Search.
Seriously. Also, lack of engineering talent. I'm just one person who has hand built
this massive platform and product and three years into the project, I am still 
working on this thing solo with not a helping hand in sight. A similar implementation
that I could consider using for this application, and perhaps this project can
one day be forked and this approach be used, would make the runtime of the application
much faster with less resources, BUT swap out that burden up front for slower per-action
requests. Meaning, when using the Project Apario application, you'll have searches that
won't be using a massive cache buffer on them, thus giving you the illusion of the site
being super fast to use. Instead, Using Go's concurrency runtime, you'd effectively be
opening/closing all of the file accessors for each of the 
`database/<collection>/<sha512checksum>/pages/ocr.######.txt` files scanning for matches.
The other option of course would be to introduce a database to the service, but then THAT
too will take up just as much, if not more resources than this current runtime 
implementation. The other aspect that I am actively keeping track of is the fact that
since this project is written in Go, it'll be relatively stable and require less hand
holding than your typical Rails application. Less moving parts and a compiled source 
code means that compiled binaries, while they will be pretty large in size, will continue
to run years into the future with little to no updates required to the application. 
Most pieces of software are constantly evolving over time for various purposes and this
project will more than likely follow that suit, but each of the compiled releases will
continue to be useable to the end-user should they so choose. Meaning, you can run an
old copy of the code without being FORCED to upgrade the software. The long and tall of 
what this means is that this project is FAR MORE sustainable for long-term usage than
the first implementation of Project Apario which also means that as new data sets get
released by the various Governments around the world, you'll be able to use the 
Apario Contribution project to compile the database and the Apario Reader to consume that
data. The STARGATE files are only the sample set being used for development since they are 
massive and awesome and will make for a great piece of software in the long run.

## Magical Effects

There is a quantum majestic magic in this repository. Sure you can dismiss this for being
nonsense or supernatural witchcraft, but the project has been centered around special
indexing that leverages gematria. This means that the magic behind gematria and the
use cases that it has for searching/researching can be fully leveraged in the collection
that you are researching.

## What this application does.

Once the application has booted, you'll be able to navigate to `https://locahost:8080` and
the GUI for Project Apario will be presented to you, where you can perform full text 
advanced search against the repository of your choosing. 

https://my.projectapario.com:8080/

### Advanced Search

Part of this code base includes a test suite for **Advanced Search** which is a unique
searching experience offered by Project Apario. Typical search engines use complex
algorithms to determine the relevancy of keywords, accuracy of terms, and matchability
of results based on what is provided as the search input, and then the results are 
condensed into manageable expectations. This works for most applications, but does not
work for services like Project Apario, and are considered counter-intuitive to the 
core philosophy behind the project. Therefore, when searching for data, this mechanism
is used instead. The test suite specifically parses the following types of queries. 

```go
qs = map[uint]string{
    Q1:  "(top secret or confidential) and communist and oswald not thought",
    Q2:  "top secret and oswald",
    Q3:  "top secret and communist not oswald",
    Q4:  "(top secret or confidential or classified) and (assassin or murder or kill) and (kennedy or president) and (communi or infiltrat) not (cover page or blank page or unclassified)",
    Q5:  "top secret and communist and not oswald",
    Q6:  "[communism,communist] and (top secret , confidential) && {communist} not kevin bacon",
    Q7:  "(top secret or confidential or classified) and (communist or communism or commie) and (assassinated or killed or died or murdered)",
    Q8:  "(top secret or confidential or classified) and (communist or communism or commie) not (assassinated or killed or died or murdered) and bacon not sausage and lettuce not mustard",
    Q9:  "(top secret or confidential or classified) && (communist or communism or commie) !(assassinated or killed or died or murdered) !mustard !sausage && bacon && lettuce",
    Q10: "(top secret,confidential , classified) && (communist,communism,commie) !(assassinated|killed||died,murdered) !mustard !sausage && bacon && lettuce",
    Q11: "(secret or confidential or classified) not (cover or intentionally left blank) and (President Kennedy or John F Kennedy or President JFK or POTUS JFK or POTUS 35)",
    Q12: "(orange juice or coffee or apple juice or tomato juice) and (sunny side up or over easy or scrambled or omelet) not alcohol and jesus and (toast or fruit bowl or french crepe)",
}
```

As you can see, the queries that can be structured will give you significant control over
what you are able to process out of the full text search results from the Apario
Contribution generated output. Since OCR isn't 100% accurate and the actual percentage
of accuracy on the per-page analysis performed, you're going to see roughly 71% accuracy.
This means, that you may need to perform searches like: 

```go
query := "(function or funclion or fuuction)" 
```

To properly capture each of the improperly formed optical character recognized words 
that come out of the records. The reason for this is the Project Apario application
was designed to ingest declassified Government records, and since the governments
of the world love to cover up their secrets and hide the truth from their public, the
records are usually very old and quite damaged making the efficiency of the optical 
character recognition (OCR) [via tesseract] only about 71% accurate. Nevertheless, 
while AI systems and "intelligent search platforms" like Elasticsearch work for clean
data, they do not work at all for declassified data; thus why Advanced Search was 
invented. 

## Security Conscious

One of the files in this project is called ssl.go and its primary responsibility is to 
provide HTTPS encrypted traffic between YOUR BROWSER and YOUR COPY of Apario Reader.
It's kind of moot at the end of the day, but every time the application is booted, you
have the option of automatically generating a self-signed SSL certificate that the GIN
web server will use to serve up HTTPS traffic for my.projectapario.com:8080 or 
localhost:8080. Otherwise, you can specify the paths to the SSL Certificate files and
not be required to restart your application when you refresh them. As a configurable
to the application, the TTL of the TLS certificate is defined and if paths are provided,
they are reloaded into the application every TTL-seconds. This means that, unlike using
Apache or Nginx to serve your TLS connection, the Apario Reader will automatically accept
the new TLS certificate and reload it at runtime for you. Replace the .crt, .ca-bundle.crt,
and .key files every year [thanks Apple] and you won't be required to run apache2ctl reload
or systemd restart nginx; thus interrupting your research.

## Gematria

This project uses Gematria. What is Gematria? Every letter in the English language is
assigned a numerical value. Depending on the flavor of Gematria that is being used,
the sum of the letters (whether its a word, a full text document or a title) gets a 
score. That score is represented as 3 numbers; English, Jewish and Simple. Gematria is
used to show you that: 

Gematria of: **manifesting three six nine**

| Code    | Value |
|---------|-------|
| English | 1602  |
| Jewish  | 1028  |
| Simple  | 267   | 

These 3 numbers can then be used for looking at other possible values. For instance,
the English Gematria value of 1602 can also represent: 

| Value | String                             |
|-------|------------------------------------|
| 1602  | Gods Children Are Not For Sale     |
| 1602  | Before I Made You I Knew You       |
| 1602  | Another One Bites The Dust         |
| 1602  | Twelve Days Of Christmas           |
| 1602  | The Magician And Nine Of Pentacles |
| 1602  | The Search For Lost Friends        |
| 1602  | A Turning Point For America        | 
| 1602  | I Am Q Without Dispute Q           |

As you can see, the possibilities for finding something else that may or may not be
considered interesting TO YOU, may come out of simple Gematria related searches. 
As a consequence of these coincidences, Project Apario has decided that in the 
Apario Reader application to compile each and every one of the full text pages,
words and titles of every single page of every single document inside the 
`database/<collection>` into Gematria values. When searching for terms, the
Gematria value of the words will be included, including the sum of the search
queries Gematria value and from those values, you can see all of the other pages
in the collection that also have that same Gematria value. 

With great power comes great responsibility, my friends. Use it well. 

## About Project Apario

Project Apario was started spring 2020 after the COVID-19 pandemic. An individual,
who at the time was unknown to almost everyone on the internet, went by the name
of Austin Steinbart, reached out to me and asked if I wanted to build a search engine
for the JFK files. I agreed to take what I had started in Fall 2019 called "Crowd
Sourcing Declas" and rebrand it as PhoenixVault. I built a new design on top of the
bare bones CSD Rails 5 app that I built and ingested the JFK files from 2018 into them.
I originally wanted to dig into these files since I have always been a huge fan of JFK.
However the National Archives released them unsearchable and I wasn't going to let that
stop me from bypassing the fake narratives and grifting books that claim to understand
what happened to President Kennedy. In Summer 2020 I launched the site and holy shit did
I get an onslaught of attacks from so many clowns you can hardly imagine. The site was 
pulled offline after Austin Steinbart was caught. He was a fraud and deceived a lot of
people, myself included, but that didn't end the project. In September 2020 I rebranded
the Rails 5 application from Crowd Sourcing Declas to PhoenixVault to Project Apario. 
The name Project Apario was inspired, not from anybody by the name of Apario, but from
the latin word Aperio which means to reveal. I'm also a huge Harry Potter nerd and I 
love me some Hermione Granger, and she was infamously known for saying "it's leviOSAH"
nor "levOSAR". So, in that spirit, I name it apARio not apERio. Throughout fall 2020
and spring 2021 I spent a majority of my spare time refactoring many parts of the 
project by removing Elasticsearch and MongoDB Atlas from the technology stack. That
refactor took me a long time and by summer 2021 I relaunched the platform as Project
Apario and by fall 2021 I had launched Advanced Search, however that only let you
use "term1 AND term2" up to 6 terms max. No OR or NOT supported at the time, AND that
required 768GB of RAM on the MongoDB cluster since it was using Regular Expressions
to provide that full text search. I found, through using the Advanced Search, that
it was a lot easier for me to understand and grok the JFK files and that really helped
give me closure to understand what happened to my favorite President growing up. In the
spring of 2023 I made the decision that the operational cost of running a nearly 1TB of
RAM Rails 5 application was no longer economical for me to do, and I discontinued that
version of the application. I developed the Apario Contribution service and made it 
open source, unlike the Rails 5 version of Project Apario. The Rails 5 version of
Project Apario used commercial products in it and therefore couldn't be released in 
its full open source format such that you'd be able to run a copy of it yourself. 
Granted than it cost me well over $3K/month to operate and required me to custom build
a private enterprise cloud in order to efficiently run the darn thing, the code wasn't
my best and the use of the commercial gems made the decision easy for me to discontinue
that version of the application. This Go based version is purely open source and is 
provided to you free of charge under the GPL-3 license. 

## Special Thanks

I want to personally thank all of the amazing supporters of the Project Apario
effort that have contributed their hard earned money, time and resources into helping
the idea spread far and wide. Throughout its tenure, hundreds of thousands of researchers
worldwide used Project Apario anonymously to search a massive database of archives. While
I am sad that the site is and has been offline for over half a year now, to date, I am grateful
that the burden of operating it single-handedly has been lifted from me, and my hopes that 
this rewrite in Go will make it stable for the long term without burdening me with massive
operational expenses to run. Instead of me bringing back a version of this software that I 
wrote, and footing most of the bill while asking others to contribute voluntarily, making it
completely free online gives everyone the best of both worlds. If somebody rich wants to help
me, they can sponsor my Github account for as little or as much as they wish. 

## Sponsorships

Project Apario will never compromise its integrity for money, that's why this is free of charge.
However, if somebody who does have a lot of money would like to see the return of Project
Apario, this offering gives you the ability to get your own copy of the software up and running
and you can pay for the servers yourself. 

## Production Environments

When considering running Project Apario in the public cloud on your own dime using this software,
keep in mind of a few things: 

1. Going with a hosting provider that gives you unmetered bandwidth will be important.
2. You'll want at least 64GB of RAM on each host.
3. You'll want to have at least 2 (min) hosts in each availability zone.
4. You'll want to have servers on the West Coast, East Coast, Central Europe and Australia.
5. You'll want to have load balancers that are geographically aware of where incoming requests originate.
6. You'll want to have at least 10TB of disk space for the JFK files, 200GB for the STARGATE files, and plenty of room for logs.

## Warranty and Liability

Read the LICENSE to fully understand the conditions of using this software.

