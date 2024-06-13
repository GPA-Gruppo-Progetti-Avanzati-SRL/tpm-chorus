# Test-01

## Open Api Repo Example 01

```
curl --request POST \
  --url http://localhost:8080/test/test01/api/v1/ep01/test \
  --header 'Content-Type: application/json' \
  --header 'canale: APBP' \
  --header 'requestId: 123456789' \
  --data '{
    "canale": "APBP",
    "dataOperazione": "20240528",
    "ordinante": {
        "natura": "DR",
        "tipologia": "ALIAS",
        "numero": "123456",
        "codiceFiscale": "SSSMMM55F28E345Z",
        "intestazione": "Dario Intesta"
    },
    "additionalProperties": {
        "additionalProp1": {},
        "additionalProp2": {},
        "additionalProp3": {}
    }
}'
```

E' necessario creare il topic `test.tpm-symphony` utilizzato in `kafka01`. Di seguito uno snippet per ambiente MacOs.

```
/Library/Kafka/bin/kafka-topics.sh --create --if-not-exists --topic test.tpm-symphony --bootstrap-server localhost:9092
```
