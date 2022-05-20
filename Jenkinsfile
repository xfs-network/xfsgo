pipeline {
    agent any
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
        stage('BuildAndInstall') {
            when {
                branch 'develop'
            }
            steps {
                updateGitlabCommitStatus name: 'BuildAndInstall', state: 'pending'
                sh """
                make
                cp ./xfsgo /opt/xfsgo/bin
                """
            }
            post {
                success {
                    updateGitlabCommitStatus name: 'BuildAndInstall', state: 'success'
                }
                failure {
                    updateGitlabCommitStatus name: 'BuildAndInstall', state: 'failed'
                }
	        }
        }
    }
}
