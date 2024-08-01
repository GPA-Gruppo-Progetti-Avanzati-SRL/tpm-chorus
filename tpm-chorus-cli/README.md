# Esecuzione di una orchestrazione

```shell
./tpm-chorus-cli orchestration \
      --orc orchestration-examples/movies \
      --req orchestration-examples/movies-orchestration-request.yaml \
      --cfg orchestration-examples/orchestrations-env-config.yaml
```

| Orchestrazione |                                                                                                             |
|----------------|-------------------------------------------------------------------------------------------------------------|
| --orc          | path relativo dove si trovano le configurazioni della orchestrazione                                        |
| --req          | definizione della richiesta in  formato yaml. La proprietà body che definisce il payload e' in formato json |
| --cfg          | configurazione delle metriche e dei linked services.                                                        |


## Struttura orchestrazione di esempio

### Body della chiamata

Cio' che viene usato e' l'elemento chiave. Il corpo non viene usato. Riportato come non rilevante per completezza.
(nota: nel caso di tpm-rhapsody i messaggi in ingresso vengono riportati ad un singolo body strutturato con le proprietà `key` e `body`)
```json
{
  "key": {
    "year": 1939,
    "title": "The Wizard of Oz"
  },
  "body": {
    "note": "not relevant"
  }
}
```

```mermaid
sequenceDiagram
    autonumber
    participant chorus as Workflow Executor
    participant sa as Start Activity
    participant ma_fo as Mongo Find One
    participant ep_a as Endpoint Get One
    participant ma_ro as Mongo Replace One
    participant ma_uo as Mongo Update One
    participant ma_ao as Mongo Aggregate One
    participant mongodb as MongoDB
    participant restsvc as Rest Service
    participant sub_a as Nested Activity
    participant ea as End Activity

	chorus ->> sa: start
    chorus ->> ma_fo: 
    ma_fo ->> mongodb: Find One
    rect rgb(224,224,224)
        note right of chorus: if not found
    chorus ->> ep_a: 
    ep_a ->> restsvc: GetOneByYearTitle()
    chorus ->> ma_ro: 
    ma_ro ->> mongodb: MongoDb op replace one
    chorus ->> ma_uo: 
    ma_uo ->> mongodb: MongoDb op update one
        note right of ma_uo: viene aggiornato un documento diverso in maniera fittizia sulla stessa collection
    end
    chorus ->> ma_ao: 
    ma_ao ->> mongodb: MongoDb op find one using aggregation framework
    chorus ->> sub_a: execute (no-op nested activity (tbd))
    chorus ->> ea: End    
```