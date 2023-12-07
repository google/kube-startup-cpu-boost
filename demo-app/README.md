# spring-demo-app

The folder contains source code of a simple Java application that uses Spring Boot framework
and exposes the REST endpoint. The data is fetched from the relational database with
Java Persistence API.

## Usage

The service exposes below endpoints:

* `/books` returns collection of books

In the addition to above, Spring Boot actuator endpoints are exposed per configuration.

### Source code

Prerequisites:

* [JDK](https://openjdk.org/projects/jdk/17/) 17 or newer
* [Maven](https://maven.apache.org/download.cgi)

Build application from the source code.

```sh
mvn package
java -jar target/spring-demo-app-0.0.1-SNAPSHOT.jar
```

### Container image

1. Adjust `application.yaml` according to your needs
2. Run the container
  
   ```sh
   docker run -d  --name spring-demo-app \
      -p 8080:8080 \
      -v "`pwd`/application.yaml:/application.yaml" \
      ghcr.io/google/spring-demo-app:latest 
   ```

### Kubernetes

1. Adjust `application.yaml` according to your needs
2. Adjust `kustomization.yaml` according to your needs
3. Customize and apply Kubernetes manifests

   ```sh
   kubectl kustomize | kubectl apply -f
   ```
