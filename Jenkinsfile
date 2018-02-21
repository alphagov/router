#!/usr/bin/env groovy

REPOSITORY = 'router'

repoName = JOB_NAME.split('/')[0]

node ('mongodb-2.4') {
  env.REPO      = 'alphagov/router'
  env.BUILD_DIR = '__build'
  env.GOPATH    = "${WORKSPACE}/${BUILD_DIR}"
  env.SRC_PATH  = "${env.GOPATH}/src/github.com/${REPO}"

  def govuk = load '/var/lib/jenkins/groovy_scripts/govuk_jenkinslib.groovy'

  try {
    stage("Checkout") {
      checkout scm
    }

    stage("Setup build environment") {
      // Clean GOPATH: Recursively delete everything in the current directory
      dir(env.GOPATH) {
        deleteDir()

        // Create Build-Path
        sh "mkdir -p ${env.SRC_PATH}"
    }

      // Seed Build-Path
      dir(env.WORKSPACE) {
        sh "/usr/bin/rsync -a ./ ${env.SRC_PATH} --exclude=$BUILD_DIR"
      }
    }

    // Start Build
    stage("Build") {
      dir(env.SRC_PATH) {
        sh 'BINARY=$WORKSPACE/router make clean build'
      }
    }

    // Run tests
    wrap([$class: 'AnsiColorBuildWrapper']) {
      stage("Test") {
        dir(env.SRC_PATH) {
          sh 'BINARY=$WORKSPACE/router make test'

          sh '$WORKSPACE/router -version'
        }
      }
    }

    // Archive Binaries from build
    stage("Archive Artifact") {
      archiveArtifacts 'router'
    }

    // Push the Go binary for the build to S3, for AWS releases
    if (env.BRANCH_NAME == "master") {
      stage("Push binary to S3") {
        govuk.uploadArtefactToS3('router', "s3://govuk-integration-artefact/router/release/router")
        target_tag = "release_${env.BUILD_NUMBER}"
        govuk.uploadArtefactToS3('router', "s3://govuk-integration-artefact/router/${target_tag}/router")
      }
    }

    if (govuk.hasDockerfile()) {
      stage("Build Docker image") {
        govuk.buildDockerImage(repoName, env.BRANCH_NAME)
      }

      stage("Push Docker image") {
        govuk.pushDockerImage(repoName, env.BRANCH_NAME)
      }

      if (env.BRANCH_NAME == "master") {
        stage("Tag Docker release image") {
          govuk.pushDockerImage(repoName, env.BRANCH_NAME, "release")
        }

        stage("Tag Docker release_${env.BUILD_NUMBER} image") {
          dockerTag = "release_${env.BUILD_NUMBER}"
          govuk.pushDockerImage(repoName, env.BRANCH_NAME, dockerTag)
        }
      }
    }

    if (env.BRANCH_NAME == "master") {
      stage("Push release tag") {
        govuk.pushTag('router', env.BRANCH_NAME, 'release_' + env.BUILD_NUMBER)
      }

      stage("Push to Gitlab") {
        try {
          govuk.pushToMirror('router', env.BRANCH_NAME, 'release_' + env.BUILD_NUMBER)
        } catch (e) {
        }
      }

      stage("Deploy to integration") {
        govuk.deployIntegration('router', env.BRANCH_NAME, "release_${env.BUILD_NUMBER}", 'deploy')
      }
    }
  } catch (e) {
      currentBuild.result = "FAILED"
      step([$class: 'Mailer',
            notifyEveryUnstableBuild: true,
            recipients: 'govuk-ci-notifications@digital.cabinet-office.gov.uk',
            sendToIndividuals: true])
    throw e
    }

}
