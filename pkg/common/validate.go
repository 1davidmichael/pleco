package common

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
)

func isUsed(cmd *cobra.Command, serviceName string) bool {
	service, err := cmd.Flags().GetBool("enable-" + serviceName)
	if err == nil && service {
		return true
	}
	return false
}

func CheckEnvVars(cloudProvider string, cmd *cobra.Command) {
	var requiredEnvVars []string

	switch cloudProvider {
	case "aws":
		// requiredEnvVars = append(requiredEnvVars, checkAWSEnvVars(cmd)...)
		// Do nothing
	case "scaleway":
		requiredEnvVars = append(requiredEnvVars, checkScalewayEnvVars(cmd)...)
	case "do":
		requiredEnvVars = append(requiredEnvVars, checkDOEnvVars(cmd)...)
	default:
		log.Fatalf("Unknown cloud provider: %s. Should be \"aws\", \"scaleway\" or \"do\"", cloudProvider)
	}

	kubeConn, err := cmd.Flags().GetString("kube-conn")
	if err == nil && kubeConn == "out" {
		requiredEnvVars = append(requiredEnvVars, "KUBECONFIG")
	}

	for _, envVar := range requiredEnvVars {
		if _, ok := os.LookupEnv(envVar); !ok {
			log.Fatalf("%s environment variable is required and not found", envVar)
		}
	}
}

func checkAWSEnvVars(cmd *cobra.Command) []string {
	// if metadata endpoint is available assume env vars are not required
	// https://stackoverflow.com/questions/44193262/how-to-identify-whether-my-container-is-running-on-aws-ecs-or-not
	// http://169.254.169.254/latest/meta-data/
	var requiredEnvVars []string
	client := &http.Client{
		Timeout: 1 * time.Second,
	}

	resp, err := client.Get("http://169.254.169.254/latest/meta-data/")
	if resp.StatusCode == 200 && err == nil {
		return []string{}
	} else {
		if isUsed(cmd, "rds") ||
			isUsed(cmd, "documentdb") ||
			isUsed(cmd, "elasticache") ||
			isUsed(cmd, "eks") ||
			isUsed(cmd, "elb") ||
			isUsed(cmd, "vpc") ||
			isUsed(cmd, "s3") ||
			isUsed(cmd, "ebs") ||
			isUsed(cmd, "cloudwatch-logs") ||
			isUsed(cmd, "kms") ||
			isUsed(cmd, "iam") ||
			isUsed(cmd, "ssh-keys") ||
			isUsed(cmd, "ecr") {

			requiredEnvVars = append(requiredEnvVars, "AWS_ACCESS_KEY_ID")
			requiredEnvVars = append(requiredEnvVars, "AWS_SECRET_ACCESS_KEY")
			return requiredEnvVars
		}
	}

	return []string{}
}

func checkScalewayEnvVars(cmd *cobra.Command) []string {
	var requiredEnvVars = []string{
		"SCW_ACCESS_KEY",
		"SCW_SECRET_KEY",
	}

	if isUsed(cmd, "cluster") ||
		isUsed(cmd, "db") ||
		isUsed(cmd, "s3") ||
		isUsed(cmd, "cr") ||
		isUsed(cmd, "lb") ||
		isUsed(cmd, "sg") ||
		isUsed(cmd, "volume") {
		return requiredEnvVars
	}

	return []string{}
}

func checkDOEnvVars(cmd *cobra.Command) []string {
	var requiredEnvVars = []string{
		"DO_API_TOKEN",
		"DO_SPACES_KEY",
		"DO_SPACES_KEY",
	}

	if isUsed(cmd, "cluster") ||
		isUsed(cmd, "db") ||
		isUsed(cmd, "s3") ||
		isUsed(cmd, "lb") ||
		isUsed(cmd, "volume") ||
		isUsed(cmd, "firewall") ||
		isUsed(cmd, "vpc") {
		return requiredEnvVars
	}

	return []string{}
}
