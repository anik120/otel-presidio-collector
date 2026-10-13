# data-anonymizer HTTP service for the collector's presidio processor.
# The data-anonymizer library is installed directly from its GitLab repo.
FROM python:3.12-slim

WORKDIR /app

# git to clone the library; build-essential covers transitive deps without a wheel.
RUN apt-get update \
    && apt-get install -y --no-install-recommends git build-essential \
    && rm -rf /var/lib/apt/lists/*

# Install data-anonymizer (pulls presidio + en_core_web_lg).
# gitlab.cee.redhat.com is signed by the Red Hat internal CA, which this base
# image does not trust. For the PoC, git cert verification is disabled for the
# clone. For a real image, install the Red Hat IT CA into the trust store
# (COPY the CA .pem into /usr/local/share/ca-certificates && update-ca-certificates)
# instead of bypassing verification.
ARG DATA_ANONYMIZER_REPO=https://gitlab.cee.redhat.com/uxe-data-ai-solutions/data-anonymizer.git
RUN GIT_SSL_NO_VERIFY=true pip install --no-cache-dir "git+${DATA_ANONYMIZER_REPO}"

# The HTTP wrapper.
RUN pip install --no-cache-dir fastapi "uvicorn[standard]"
COPY app.py /app/app.py

EXPOSE 8000
CMD ["uvicorn", "app:app", "--host", "0.0.0.0", "--port", "8000"]
