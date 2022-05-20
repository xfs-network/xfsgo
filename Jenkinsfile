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
        stage('Build') {
            when {
                branch 'develop'
            }
            steps {
                sh """
                make
                """
            }
            post {
                success {
                    updateGitlabCommitStatus name: 'Build', state: 'success'
                }
                failure {
                    updateGitlabCommitStatus name: 'Build', state: 'failed'
                }
	        }
        }
    }
}
