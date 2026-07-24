pipeline {
    agent any
    environment {
        DOCKER_REGISTRY = "192.168.74.68:5000"
        GEMINI_API_KEY = credentials("gemini-api-key")
        SLACK_WEBHOOK = credentials("slack-webhook")
    }
    stages {
        stage("Checkout") {
            steps {
                checkout scm
            }
        }
        stage("SonarQube Scan") {
            steps {
                script {
                    // SonarQube scan for front-end (Node.js)
                    dir("front-end") {
                        def scannerHome = tool 'SonarQube'
                        withSonarQubeEnv("SonarQubeServer") {
                            sh "${scannerHome}/bin/sonar-scanner -Dsonar.projectKey=sockshop-front-end -Dsonar.sources=."
                        }
                    }
                    // SonarQube scan for Go services (catalogue, user, payment) - requires Go plugin for SonarQube
                    // For simplicity, we'll skip detailed Go SonarQube analysis in this example, focusing on image scanning.
                    // In a real scenario, you would configure SonarQube to analyze Go projects.
                }
            }
        }
        stage("Build and Scan Front-End") {
            steps {
                script {
                    dir("front-end") {
                        sh "docker build -t ${DOCKER_REGISTRY}/front-end:latest ."
                        sh "docker push ${DOCKER_REGISTRY}/front-end:latest"
                        sh "trivy image --severity CRITICAL ${DOCKER_REGISTRY}/front-end:latest > ../trivy-front-end.txt"
                    }
                }
            }
        }
        stage("Build and Scan Catalogue") {
            steps {
                script {
                    dir("catalogue") {
                        sh "docker build -t ${DOCKER_REGISTRY}/catalogue:latest -f docker/catalogue/Dockerfile ."
                        sh "docker push ${DOCKER_REGISTRY}/catalogue:latest"
                        sh "trivy image --severity CRITICAL ${DOCKER_REGISTRY}/catalogue:latest > ../trivy-catalogue.txt"
                    }
                }
            }
        }
        stage("Build and Scan User") {
            steps {
                script {
                    dir("user") {
                        sh "docker build -t ${DOCKER_REGISTRY}/user:latest ."
                        sh "docker push ${DOCKER_REGISTRY}/user:latest"
                        sh "trivy image --severity CRITICAL ${DOCKER_REGISTRY}/user:latest > ../trivy-user.txt"
                    }
                }
            }
        }
        stage("Build and Scan Payment") {
            steps {
                script {
                    dir("payment") {
                        sh "docker build -t ${DOCKER_REGISTRY}/payment:latest -f docker/payment/Dockerfile ."
                        sh "docker push ${DOCKER_REGISTRY}/payment:latest"
                        sh "trivy image --severity CRITICAL ${DOCKER_REGISTRY}/payment:latest > ../trivy-payment.txt"
                    }
                }
            }
        }
        stage("Deploy to Kubernetes" ) {
            steps {
                script {
                    sh '''
                        export KUBECONFIG=/var/lib/jenkins/.kube/config
                        echo "Deploying to Kubernetes cluster at $(kubectl config view --minify --output 'jsonpath={..server}')..."
                        kubectl apply -f front-end-deployment.yaml
                        kubectl apply -f catalogue-deployment.yaml
                        kubectl apply -f catalogue-db-deployment.yaml
                        kubectl apply -f user-deployment.yaml
                        kubectl apply -f user-db-deployment.yaml
                        kubectl apply -f payment-deployment.yaml
                        # Force pods to restart and pull latest images
                        kubectl rollout restart deployment/front-end
                        kubectl rollout restart deployment/catalogue
                        kubectl rollout restart deployment/catalogue-db
                        kubectl rollout restart deployment/user
                        kubectl rollout restart deployment/user-db
                        kubectl rollout restart deployment/payment
                    '''
                }
            }
        }
    }
    post {
        failure {
            script {
                // Concatenate all Trivy reports for AI analysis
                // sh "cat trivy-front-end.txt trivy-catalogue.txt trivy-user.txt trivy-payment.txt > combined-trivy-report.txt"
                // sh "tail -n 200 combined-trivy-report.txt | python3 ai/analyzer.py"
                sh "cat trivy-front-end.txt > combined-trivy-report.txt 2>/dev/null || echo 'No trivy report' > combined-trivy-report.txt"
                sh "cat trivy-catalogue.txt >> combined-trivy-report.txt 2>/dev/null || true"
                sh "cat trivy-user.txt >> combined-trivy-report.txt 2>/dev/null || true"
                sh "cat trivy-payment.txt >> combined-trivy-report.txt 2>/dev/null || true"
                // Then run AI analysis
                sh "tail -n 200 combined-trivy-report.txt | python3 ai/analyzer.py || echo 'AI analysis skipped - quota exceeded or unavailable'"
            }
        }
    }
}
