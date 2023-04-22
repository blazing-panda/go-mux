# Go-mux

## Exercise 3: CI workflow

### Go action

I used Github actions to create a CI workflow for the project.
The workflow to build and run the tests is defined in `.github/workflows/go.yml` and runs on every push to the repository.

The workflow first sets up the postgres database needed for the tests as a service.
It uses `actions/checkout@v3` to checkout the repository and `actions/setup-go@v4` to setup the required go environment, after which it installs the dependencies with `go mod download`. Once the dependencies are installed, the project is build with `go build` and the tests are run with `go test`.

.go.yml
```yaml
# This workflow will build a golang project
# For more information see: https://docs.github.com/en/actions/automating-builds-and-tests/building-and-testing-go

name: Go

on:
  push:
    branches: [ "main" ]
  pull_request:
    branches: [ "main" ]

jobs:

  build:
    runs-on: ubuntu-latest
    env:
      APP_DB_USERNAME: postgres
      APP_DB_PASSWORD: postgres
      APP_DB_NAME: postgres
      APP_JWT_SECRET: postgres
    services:
      postgres:
        image: postgres:latest
        env:
          POSTGRES_USER: postgres
          POSTGRES_PASSWORD: postgres
          POSTGRES_DB: postgres
        ports:
          - 5432:5432
        options: --health-cmd pg_isready --health-interval 10s --health-timeout 5s --health-retries 3
    steps:
    - uses: actions/checkout@v3

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: 1.19

    - name: Install dependencies
      run: |
        cd src
        go mod download

    - name: Build
      run: |
        cd src
        go build -v

    - name: Test
      run: |
        cd src
        go test -v
```

.successfull go action
image::doc/images/image-2023-04-22-18-14-40-479.png[]

This should display the status of the workflow

[![Go](https://github.com/blazing-panda/go-mux/actions/workflows/go.yml/badge.svg)](https://github.com/blazing-panda/go-mux/actions/workflows/go.yml)

### SonarCloud action

I used the sonarcloud template to create my workflow for the sonarcloud analysis and followed all of the steps mentioned in the template.

.sonarcloud.yml
```yaml
# This workflow uses actions that are not certified by GitHub.
# They are provided by a third-party and are governed by
# separate terms of service, privacy policy, and support
# documentation.

# This workflow helps you trigger a SonarCloud analysis of your code and populates
# GitHub Code Scanning alerts with the vulnerabilities found.
# Free for open source project.

# 1. Login to SonarCloud.io using your GitHub account

# 2. Import your project on SonarCloud
#     * Add your GitHub organization first, then add your repository as a new project.
#     * Please note that many languages are eligible for automatic analysis,
#       which means that the analysis will start automatically without the need to set up GitHub Actions.
#     * This behavior can be changed in Administration > Analysis Method.
#
# 3. Follow the SonarCloud in-product tutorial
#     * a. Copy/paste the Project Key and the Organization Key into the args parameter below
#          (You'll find this information in SonarCloud. Click on "Information" at the bottom left)
#
#     * b. Generate a new token and add it to your Github repository's secrets using the name SONAR_TOKEN
#          (On SonarCloud, click on your avatar on top-right > My account > Security
#           or go directly to https://sonarcloud.io/account/security/)

# Feel free to take a look at our documentation (https://docs.sonarcloud.io/getting-started/github/)
# or reach out to our community forum if you need some help (https://community.sonarsource.com/c/help/sc/9)

name: SonarCloud analysis

on:
  push:
    branches: [ "main" ]
  pull_request:
    branches: [ "main" ]
  workflow_dispatch:

permissions:
  pull-requests: read # allows SonarCloud to decorate PRs with analysis results

jobs:
  Analysis:
    runs-on: ubuntu-latest

    steps:
      - name: Analyze with SonarCloud

        # You can pin the exact commit or the version.
        # uses: SonarSource/sonarcloud-github-action@de2e56b42aa84d0b1c5b622644ac17e505c9a049
        uses: SonarSource/sonarcloud-github-action@de2e56b42aa84d0b1c5b622644ac17e505c9a049
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}  # Needed to get PR information
          SONAR_TOKEN: ${{ secrets.SONAR_TOKEN }}   # Generate a token on Sonarcloud.io, add it to the secrets of this repo with the name SONAR_TOKEN (Settings > Secrets > Actions > add new repository secret)
        with:
          # Additional arguments for the sonarcloud scanner
          args:
            # Unique keys of your project and organization. You can find them in SonarCloud > Information (bottom-left menu)
            # mandatory
            -Dsonar.projectKey=blazing-panda_go-mux
            -Dsonar.organization=blazing-panda
            # Comma-separated paths to directories containing main source files.
            #-Dsonar.sources= # optional, default is project base directory
            # When you need the analysis to take place in a directory other than the one from which it was launched
            #-Dsonar.projectBaseDir= # optional, default is .
            # Comma-separated paths to directories containing test source files.
            #-Dsonar.tests= # optional. For more info about Code Coverage, please refer to https://docs.sonarcloud.io/enriching/test-coverage/overview/
            # Adds more detail to both client and server-side analysis logs, activating DEBUG mode for the scanner, and adding client-side environment variables and system properties to the server-side log of analysis report processing.
            #-Dsonar.verbose= # optional, default is false
```

.sonarcloud action
image::doc/images/sonarcloud-action.png[]

[![SonarCloud analysis](https://github.com/blazing-panda/go-mux/actions/workflows/sonarcloud.yml/badge.svg)](https://github.com/blazing-panda/go-mux/actions/workflows/sonarcloud.yml)

.sonarcloud.png
image::doc/images/sonarcloud.png[]


## Setup

For running this application you need to have docker installed and fire up a postgres database with this command:

 docker run -it -p 5432:5432 -e POSTGRES_HOST_AUTH_METHOD=trust -d postgres

Following this, you should set up the following environment variables:

 export APP_JWT_SECRET=postgres
 export APP_DB_USERNAME=postgres
 export APP_DB_PASSWORD=
 export APP_DB_NAME=postgres

The test can be run via:

 go test -v
