@ PLAX: Plex Local Asset eXporter.

Export [Plex](https://plex.tv) posters, covers, and backgrounds to the corresponding media directory.

## Quickstart

1. Install `plax` on your [Plex](https://plex.tv) server (requires [Go](https://go.dev)).

    ```sh
    go install github.com/blahspam/plax 
    ```

2. Find your [Plex Authentication Token](https://support.plex.tv/articles/204059436-finding-an-authentication-token-x-plex-token/)

3. Run `plax`, replacing `plex_auth_token` with the actual token from Step 2

    ```sh
    plax -u http://127.0.0.1:32400 -t plex_auth_tokent 
    ```

## Advanced Usage 

A Docker image is available for more advanced use cases 

### Docker Compose

This example allows exporting from a remote system but requires that library
directories are mounted and writeable by `$PUID:$PGID`. 

**Example:**

```yanl
# docker-compose.yaml
plax:
  image: ghcr.io/blahspam/plax:latest
  env:
    PUID: 1000
    PGID: 1000
    PLEX_URL: http://192.0.2.1:32400
    PLEX_TOKEN: secret_plex_token
  volumes:
    - /mnt/media/movies:/movies
    - /mnt/media/tv:/tv 
```

### Kubernetes CronJob

Example of a Kubernetes CronJob 

```yaml
---
apiVersion: batch/v1
kind: CronJob
metadata:
  name: plax
  namespace: default
spec:
  schedule: "@daily"
  successfulJobsHistoryLimit: 1
  failedJobsHistoryLimit: 3
  concurrencyPolicy: Forbid
  jobTemplate:
    spec:
      template:
        spec:
          automountServiceAccountToken: false
          restartPolicy: OnFailure
          containers:
            - name: plax-container
              image: ghcr.io/blahspam/plax:1.0.0
              volumeMounts:
                - name: media
                  mountPath: /mnt/media
              env:
                - name: PUID
                  value: "1001"
                - name: PGID
                  value: "1001"
                - name: PLEX_URL
                  value: "http://192.0.2.0:32400"
                - name: PLEX_TOKEN
                  valueFrom:
                    secretKeyRef:
                      name: plex-token
                      key: token
          volumes:
            - name: media
              nfs:
                path: /mnt/media/
                server: nas.local
```

## Assumptions

This utility assumes your Plex library is structured according to current best
practices including:

* [Movies in Their Own Folders](https://support.plex.tv/articles/naming-and-organizing-your-movie-media-files/#toc-0)
* [Standard, Season-Based Shows](https://support.plex.tv/articles/naming-and-organizing-your-tv-show-files/#toc-0)
* [Artists with Album Subdirectories](https://support.plex.tv/articles/200265296-adding-music-media-from-folders/#toc-0)
