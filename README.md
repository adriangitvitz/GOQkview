# GOQkview

The project aims to develop a system that handles **qkview** files, which are uploaded to a MinIO object storage service. 
The system utilizes a consumer application built with Go to retrieve new messages from a Kafka topic. When a new message is received, 
the consumer application downloads the corresponding qkview file from MinIO. It then decompresses the file and saves its contents locally, using the file name as the destination directory.

### Configuration

Kafka ( .docker/kafka ):

* Create a **.env** file containing: <br>
  Max amount of memory used by JVM: **KAFKA_BROKER_HEAP_OPTS="-XX:MaxRAMPercentage=70.0"** <br>
  Used to communicate externally using MinIO: **DOCKER_HOST_IP="local or server ip"**

MinIO ( .docker/minio ):

* Create a new **Event** using Kafka Destination in the MinIO web instance ( http://localhost:9000 )
![image](https://github.com/adriangitvitz/GOQkview/assets/39295224/5ead526e-9f82-495d-90be-db37aa1eae8b)
* After creating the bucket subscribe to the event destination
  ![image](https://github.com/adriangitvitz/GOQkview/assets/39295224/817ef417-12f7-4c3c-b139-e6b4cd25951a)
