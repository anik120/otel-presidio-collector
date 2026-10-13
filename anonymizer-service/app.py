"""HTTP wrapper around the data-anonymizer library.

The collector's presidio processor calls POST /anonymize per span attribute.
The redaction policy lives here, not in the Go processor: data-anonymizer leaves
organization names in by default (so product/tech names like OpenShift and RHEL
survive), and a deny_list forces specific orgs to be redacted.
"""

import os

from data_anonymizer.anonymizer import PIIAnonymizer
from fastapi import FastAPI
from pydantic import BaseModel

# Terms that must always be redacted even though general organization detection
# is off. Override at deploy time without touching code via BLOCK_ORGS (comma-separated).
_block_orgs = [
    term.strip()
    for term in os.environ.get("BLOCK_ORGS", "Red Hat,RedHat").split(",")
    if term.strip()
]

# Loads the spaCy model once at startup. Model is selectable via
# DATA_ANONYMIZER_SPACY_MODEL (en_core_web_lg default; en_core_web_trf needs the
# transformer extra in the image).
anonymizer = PIIAnonymizer(
    deny_list={"ORGANIZATION": _block_orgs} if _block_orgs else None,
)

app = FastAPI(title="data-anonymizer service")


class AnonymizeRequest(BaseModel):
    text: str


class AnonymizeResponse(BaseModel):
    text: str


@app.get("/health")
def health() -> dict[str, str]:
    """Liveness/readiness check."""
    return {"status": "ok"}


@app.post("/anonymize", response_model=AnonymizeResponse)
def anonymize(req: AnonymizeRequest) -> AnonymizeResponse:
    """Return the input text with PII redacted by data-anonymizer."""
    return AnonymizeResponse(text=anonymizer.anonymize_text(req.text))
