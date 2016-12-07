// Invoked by Jenkinsfile

node {
    env.REPO      = 'alphagov/router'
    env.BUILD_DIR = '__build'
    env.GOPATH    = "${WORKSPACE}/${BUILD_DIR}"
    env.SRC_PATH  = "${env.GOPATH}/src/github.com/${REPO}"

    // Clean GOPATH: Recursively delete everything in current directory
         stage("Setup build environment") {
             dir(env.GOPATH) {
                 deleteDir()
             }

     // Create and Seed Build-Path
             sh "mkdir -p ${env.SRC_PATH}"

             dir(env.WORKSPACE) {
                 sh "/usr/bin/rsync -a ./ ${env.SRC_PATH} --exclude=$BUILD_DIR"  //
             }
         }

    // Start Build
         stage("Build") {
             dir(env.SRC_PATH) {
                 sh 'BINARY=$WORKSPACE/router make clean build test'
             }

             dir(env.GOPATH)
             sh './router -version'
         }
}
