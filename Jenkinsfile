pipeline {
    agent any
    environment {
        IMAGE_REPOSITORY = 'reg.docker.dsyun.io'
        IMAGE_NAME = 'xfsgo'
     }
    options {
      gitLabConnection('gitlab')
    }
    stages {
        stage('Test'){
            when {
                not { branch 'master' }
            }
            steps {
                updateGitlabCommitStatus name: 'Test', state: 'pending'
                sh """
                    go version
                    export GOPROXY=https://goproxy.io,direct
                    export GOSUMDB=off
                    make test
                    """
            }
            post {
                success {
                    updateGitlabCommitStatus name: 'Test', state: 'success'
                }
                failure {
                    updateGitlabCommitStatus name: 'Test', state: 'failed'
                }
            }
        }
        stage('BuildAndRelease') {
            when {
                branch 'develop'
            }
            steps {
                script {
                    updateGitlabCommitStatus name: 'BuildAndRelease', state: 'pending'
                    dockerImage = docker.build("${IMAGE_REPOSITORY}/${IMAGE_NAME}",
                     ".")
                    docker.withRegistry("https://${IMAGE_REPOSITORY}",
                         "reg.docker.dsyun.io"){
                            dockerImage.push()
                    }
                }
            }
            post {
                success {
                    updateGitlabCommitStatus name: 'BuildAndRelease', state: 'success'
                }
                failure {
                    updateGitlabCommitStatus name: 'BuildAndRelease', state: 'failed'
                }
	        }
        }
    }
}
