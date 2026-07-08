FROM python:3.12-slim

RUN pip install --no-cache-dir snmpsim pysmi \
    && useradd --create-home --uid 1000 snmpsim

WORKDIR /data
COPY data/ /data/
RUN chown -R snmpsim:snmpsim /data

USER snmpsim
EXPOSE 1161/udp

# Community name is derived from the .snmprec filename ("public").
CMD ["snmpsim-command-responder", \
     "--data-dir=/data", \
     "--agent-udpv4-endpoint=0.0.0.0:1161", \
     "--logging-method=stdout", \
     "--log-level=info"]
