## Scalemate.io user accounts service

### Run tests

To run all the tests:

```
drone exec -e ID_RSA=$ID_RSA -e ID_RSA_PUB=$ID_RSA_PUB
```

Where `$ID_RSA` and `$ID_RSA_PUB` are SSH deployment keys to access
https://github.com/mennanov/scalemate/shared

#### Run the app

```
docker-compose up
```

`$ID_RSA` and `$ID_RSA_PUB` must be defined to run Docker compose.