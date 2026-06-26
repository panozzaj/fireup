# Rails

```yaml
description: My Rails App
root: ~/projects/my-rails-app
cmd: bin/rails server -p $PORT
```

No extra configuration needed.

For version managers (rvm, rbenv), wrap in a login shell:

    cmd: bash -lc 'bin/rails server -p $PORT'
