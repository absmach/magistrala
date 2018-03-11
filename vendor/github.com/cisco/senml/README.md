# senmlCat
Tool to convert SenML between formats and act as gateway server to other services

# usage

## convert JSON SenML to XML 
senmlCat -json -i data.json > data.xml

## convert JSON SenML to CBOR
senmlCat.go -ijson -cbor data.json > data.cbor 

## convert to Excel spreadsheet CSV file
senmlCat -expand -ijsons -csv -print data.json > foo.csv

Note that this moves times to excel times that are days since 1900

## listen for posts of SenML in JSON and send to influxdb

This listens on port 880 then writes to an influx instance at localhost where to
the database called "junk"

The -expand is needed to expand base values into each line of the Line Protocol

senmlCat -ijsons -http 8880 -expand -linp -print -post http://localhost:8086/write?db=junk

