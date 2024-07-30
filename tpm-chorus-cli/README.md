# Esecuzione di una orchestrazione

```shell
./tpm-chorus-cli orchestration \
      --orc ../examples/movies-orchestration \
      --req orchestration-examples/movies-orchestration-request.yaml \
      --cfg orchestration-examples/orchestrations-env-config.yaml
```

| Orchestrazione |                                                                                                             |
|----------------|-------------------------------------------------------------------------------------------------------------|
| --orc          | path relativo dove si trovano le configurazioni della orchestrazione                                        |
| --req          | definizione della richiesta in  formato yaml. La proprietà body che definisce il payload e' in formato json |
| --cfg          | configurazione delle metriche e dei linked services.                                                        |
