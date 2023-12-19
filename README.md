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

When this application boots, it'll use a lot of memory. In its current iteration as of
the initial commit of the project, it uses 1.5GB of RAM. This memory usage is due to
the ingestion process that Golang will partake in to load the `database/<collection>`.
Upon building this application, I am using the *STARGATE* collection from the
[Apario Contribution](https://github.com/andreimerlescu/apario-contribution) project.

## Magical Effects

There is a quantum majestic magic in this repository. Sure you can dismiss this for being
nonsense or supernatural witchcraft, but the project has been centered around special
indexing that leverages gematria. This means that the magic behind gematria and the
use cases that it has for searching/researching can be fully leveraged in the collection
that you are researching.

## What this application does.

Once the application has booted, you'll be able to navigate to `https://locahost:3000` and
the GUI for Project Apario will be presented to you, where you can perform full text 
advanced search against the repository of your choosing. 

https://local.projectapario.com:3000/

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
query := "(function or funccion or fuuction)" 
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



