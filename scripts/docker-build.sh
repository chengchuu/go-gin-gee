#!/bin/bash
# bash ./scripts/docker-build.sh -r "WECOM_ROBOT_CHECK=b2lsjd46-7146-4nv2-8767-86cb0cncjdbe" "BASE_URL=https://example.com/path"

echo "Start Build Docker"

# Define command-line flags
RUN_FLAG="RUN"
ENV_VARS=""
while [[ $# -gt 0 ]]; do
  case $1 in
    -r|--run)
      RUN_FLAG="RUN"
      shift
      ;;
    -b|--build)
      RUN_FLAG="BUILD"
      shift
      ;;
    -h|--help)
      echo "Usage: docker-build.sh [OPTIONS] [ENV_VARS...]"
      echo "Build and run a Docker container for the go-gin-gee API."
      echo ""
      echo "Options:"
      echo "  -r, --run     Run the Docker container after building (default)"
      echo "  -b, --build   Build the Docker image but do not run it"
      echo "  -h, --help    Print this help message and exit"
      echo ""
      echo "Environment variables:"
      echo "  Any additional arguments passed to the script will be passed as environment variables to the Docker container."
      echo ""
      exit 0
      ;;
    *)
      ENV_VARS="$ENV_VARS -e $1"
      echo "Added environment variable: $1"
      shift
      ;;
  esac
done

echo "ENV_VARS: $ENV_VARS"

# ProjectName/SubName
preVersion="go-gin-gee/api"
# Port
visitPort="3000"
innerPort="3000"

# Stop
echo "Stop Docker Containers"
docker ps
docker ps|awk '{if (NR!=1) print $1}'| xargs docker stop

# Remove
echo "Remove Docker Containers"
docker ps -a -q
docker rm $(docker ps -a -q)

# Generate
randomVersion=${RANDOM}
combinedVersion="${preVersion}:v${randomVersion}"
echo "Generate random version: ${combinedVersion}"

# Build
echo "Build Docker Image: ${combinedVersion}"
docker build -t ${combinedVersion} . -f ./Dockerfile

# Run
if [ ${RUN_FLAG} = "RUN" ]; then
  echo "Run Docker"
  echo "Environment variables: $ENV_VARS"
  docker run --name go-gin-gee ${ENV_VARS} -d -p ${visitPort}:${innerPort} ${combinedVersion}
  # Notification
  echo "Complete, Visit: http://localhost:${visitPort}/api/ping"
else
  # Build
  # Example: 20230122153113
  # https://www.cyberciti.biz/faq/linux-unix-formatting-dates-for-display/
  DATE_FORMAT=$(date +"%Y%m%d%H%M%S")
  REPOSITORY_TAGNAME="mazeyqian/go-gin-gee:v${DATE_FORMAT}-api"
  echo "DATE_FORMAT: ${DATE_FORMAT}"
  docker tag ${combinedVersion} ${REPOSITORY_TAGNAME}
  docker push ${REPOSITORY_TAGNAME}
  echo "RUN_FLAG: ${RUN_FLAG}"
  echo "CURRENT_VERSION: v${DATE_FORMAT}"
  echo "REPOSITORY_TAGNAME: ${REPOSITORY_TAGNAME}"
  echo "All done."
fi
