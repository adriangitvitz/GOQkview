# GOQkview

The project aims to develop a system that handles **qkview** files, which are uploaded to a MinIO object storage service. 
The system utilizes a consumer application built with Go to retrieve new messages from a Kafka topic. When a new message is received, 
the consumer application downloads the corresponding qkview file from MinIO. It then decompresses the file and saves its contents locally, using the file name as the destination directory.

### Dependencies

Env variables
```shell
go get github.com/joho/godotenv
```
Kafka connection
```shell
go get github.com/Shopify/sarama
```
MinIO connection
```shell
go get github.com/minio/minio-go/v7
```
Elastic Search
```shell
go get github.com/elastic/go-elasticsearch/v8
```
UUID
```shell
go get github.com/google/uuid
```

### Configuration

Kafka ( .docker/kafka ):

* Create a **.env** file containing: <br>
  Max amount of memory used by JVM: **KAFKA_BROKER_HEAP_OPTS="-XX:MaxRAMPercentage=70.0"** <br>
  Used to communicate externally using MinIO: **DOCKER_HOST_IP="local or server ip"**

MinIO ( .docker/minio ):

* Create a **.env** file containing: <br>
  **MINIO_USER="miniouser"** <br>
  **MINIO_PASSWORD="miniopassword"**
* Create a new **Event** using Kafka Destination in the MinIO web instance ( http://localhost:9000 )
![Screenshot from 2023-06-12 01-10-17](https://github.com/adriangitvitz/GOQkview/assets/39295224/3a59d430-ccad-4196-a846-db552c1033dc)
* After creating the bucket subscribe to the event destination
![Screenshot from 2023-06-12 01-12-15](https://github.com/adriangitvitz/GOQkview/assets/39295224/7724f569-7a9b-41e7-acb7-cab48fe0c702)

ElasticSearch ( .docker/elastic ):

* Create a **.env** file containing: <br>
  **ELASTIC_PASSWORD="password"** <br>
  **MEM_LIMIT="memorylimit"**
