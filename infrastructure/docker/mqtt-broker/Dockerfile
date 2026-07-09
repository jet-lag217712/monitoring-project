FROM eclipse-mosquitto:2

COPY mosquitto.conf /mosquitto/config/mosquitto.conf
COPY acl.conf /mosquitto/config/acl.conf

# passwords and certs are mounted at runtime (see docker-compose.yaml)
EXPOSE 8883
