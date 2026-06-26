# uvicorn (FastAPI, Starlette)

```yaml
description: My API
root: ~/projects/my-api
cmd: uvicorn app:app --port $PORT
```

No extra configuration needed.

If using a virtualenv:

    cmd: .venv/bin/uvicorn app:app --port $PORT
