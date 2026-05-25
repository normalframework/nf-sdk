FROM python:3.12-slim

WORKDIR /app

ENV PYTHONUNBUFFERED=1 \
    PYTHONPATH=/app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY openfdd_nf/ ./openfdd_nf/
COPY rules/ ./rules/

# Mount mapping.yaml at runtime: -v $(pwd)/mapping.yaml:/app/mapping.yaml
ENV OPENFDD_MAPPING=/app/mapping.yaml

CMD ["python", "-m", "openfdd_nf"]
