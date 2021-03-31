package aws

import (
	"github.com/Qovery/pleco/providers/aws/database"
	ec22 "github.com/Qovery/pleco/providers/aws/ec2"
	eks2 "github.com/Qovery/pleco/providers/aws/eks"
	iam2 "github.com/Qovery/pleco/providers/aws/iam"
	"github.com/Qovery/pleco/providers/aws/logs"
	"github.com/Qovery/pleco/providers/aws/vpc"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"sync"
	"time"
)


func RunPlecoAWS(cmd *cobra.Command, regions []string, interval int64, dryRun bool, wg *sync.WaitGroup) {
	tagName, _ := cmd.Flags().GetString("tag-name")

	for _, region := range regions {
		// AWS session
		currentSession, err := CreateSession(region)
		if err != nil {
			logrus.Errorf("AWS session error: %s", err)
		}

		wg.Add(1)
		go runPlecoInRegion(cmd, region, interval, dryRun, wg, currentSession, tagName)
	}

	// AWS session
	currentSession, err := CreateSession(regions[0])
	if err != nil {
		logrus.Errorf("AWS session error: %s", err)
	}
	wg.Add(1)
	go runPlecoInGlobal(cmd, interval, dryRun, wg, currentSession, tagName)
}

func runPlecoInRegion(cmd *cobra.Command, region string, interval int64, dryRun bool, wg *sync.WaitGroup, currentSession *session.Session, tagName string) {
	defer wg.Done()

	var currentRdsSession *rds.RDS
	var currentElasticacheSession *elasticache.ElastiCache
	var currentEKSSession *eks.EKS
	var currentElbSession *elbv2.ELBV2
	var currentEC2Session *ec2.EC2
	var currentCloudwatchLogsSession *cloudwatchlogs.CloudWatchLogs
	var currentKMSSession *kms.KMS
	var currentECRSession *ecr.ECR
	elbEnabled := false
	ebsEnabled := false

	// RDS + DocumentDB connection
	rdsEnabled, _ := cmd.Flags().GetBool("enable-rds")
	documentdbEnabled, _ := cmd.Flags().GetBool("enable-documentdb")
	if rdsEnabled || documentdbEnabled {
		currentRdsSession = database.RdsSession(*currentSession, region)
	}

	// Elasticache connection
	elasticacheEnabled, _ := cmd.Flags().GetBool("enable-elasticache")
	if elasticacheEnabled {
		currentElasticacheSession = database.ElasticacheSession(*currentSession, region)
	}

	// EKS connection
	eksEnabled, _ := cmd.Flags().GetBool("enable-eks")
	if eksEnabled {
		currentEKSSession = eks.New(currentSession)
		currentElbSession = elbv2.New(currentSession)
		elbEnabled = true
		currentEC2Session = ec2.New(currentSession)
		ebsEnabled = true
		currentCloudwatchLogsSession = cloudwatchlogs.New(currentSession)
	}

	// ELB connection
	elbEnabledByUser, _ := cmd.Flags().GetBool("enable-elb")
	if elbEnabled || elbEnabledByUser {
		currentElbSession = elbv2.New(currentSession)
		elbEnabled = true
	}

	// EBS connection
	ebsEnabledByUser, _ := cmd.Flags().GetBool("enable-ebs")
	if ebsEnabled || ebsEnabledByUser {
		currentEC2Session = ec2.New(currentSession)
		ebsEnabled = true
	}

	// VPC
	vpcEnabled, _ := cmd.Flags().GetBool("enable-vpc")
	if vpcEnabled {
		currentEC2Session = ec2.New(currentSession)
	}

	// Cloudwatch
	cloudwatchLogsEnabled, _ := cmd.Flags().GetBool("enable-cloudwatch-logs")
	if cloudwatchLogsEnabled {
		currentCloudwatchLogsSession = cloudwatchlogs.New(currentSession)
	}

	// KMS
	kmsEnabled, _ := cmd.Flags().GetBool("enable-kms")
	if kmsEnabled {
		currentKMSSession = kms.New(currentSession)
	}

	// SSH
	sshEnabled, _ := cmd.Flags().GetBool("enable-ssh")
	if sshEnabled {
		currentEC2Session = ec2.New(currentSession)
	}

	// ECR
	ecrEnabled, _ := cmd.Flags().GetBool("enable-ecr")
	if ecrEnabled {
		currentECRSession = ecr.New(currentSession)
	}

	for {
		// check RDS
		if rdsEnabled {
			logrus.Debugf("Listing all RDS databases in region %s.", *currentRdsSession.Config.Region)
			database.DeleteExpiredRDSDatabases(*currentRdsSession, tagName, dryRun)
		}

		// check DocumentDB
		if documentdbEnabled {
			logrus.Debugf("Listing all DocumentDB databases in region %s.", *currentRdsSession.Config.Region)
			database.DeleteExpiredDocumentDBClusters(*currentRdsSession, tagName, dryRun)
		}

		// check Elasticache
		if elasticacheEnabled {
			logrus.Debugf("Listing all Elasticache databases in region %s.", *currentElasticacheSession.Config.Region)
			database.DeleteExpiredElasticacheDatabases(*currentElasticacheSession, tagName, dryRun)
		}

		// check EKS
		if eksEnabled {
			logrus.Debugf("Listing all EKS clusters in region %s.", *currentEKSSession.Config.Region)
			eks2.DeleteExpiredEKSClusters(*currentEKSSession, *currentEC2Session, *currentElbSession, *currentCloudwatchLogsSession, tagName, dryRun)
		}

		// check load balancers
		if elbEnabled {
			logrus.Debugf("Listing all ELB load balancers in region %s.", *currentElbSession.Config.Region)
			ec22.DeleteExpiredLoadBalancers(*currentElbSession, tagName, dryRun)
		}

		// check EBS volumes
		if ebsEnabled {
			logrus.Debugf("Listing all EBS volumes in region %s.", *currentEC2Session.Config.Region)
			ec22.DeleteExpiredVolumes(*currentEC2Session, tagName, dryRun)
		}

		// check VPC
		if vpcEnabled {
			logrus.Debugf("Listing all VPC resources in region %s.", *currentEC2Session.Config.Region)
			vpc.DeleteExpiredVPC(*currentEC2Session, tagName, dryRun)
		}

		//check Cloudwatch
		if cloudwatchLogsEnabled {
			logrus.Debugf("Listing all Cloudwatch logs in region %s.", *currentCloudwatchLogsSession.Config.Region)
			logs.DeleteExpiredLogs(*currentCloudwatchLogsSession, tagName, dryRun)
		}

		// check KMS
		if kmsEnabled {
			logrus.Debugf("Listing all KMS keys in region %s.", *currentKMSSession.Config.Region)
			DeleteExpiredKeys(*currentKMSSession, tagName, dryRun)
		}

		// check SSH
		if sshEnabled {
			logrus.Debugf("Listing all EC2 key pairs in region %s.", *currentEC2Session.Config.Region)
			ec22.DeleteExpiredKeys(currentEC2Session, tagName, dryRun)
		}

		// check ECR
		if ecrEnabled {
			logrus.Debugf("Listing all ECR repositoriesin region %s.", *currentECRSession.Config.Region)
			eks2.DeleteEmptyRepositories(currentECRSession, dryRun)
		}

		time.Sleep(time.Duration(interval) * time.Second)
	}

}

func runPlecoInGlobal(cmd *cobra.Command, interval int64, dryRun bool, wg *sync.WaitGroup, currentSession *session.Session, tagName string) {
	defer wg.Done()

	var currentS3Session *s3.S3
	var currentIAMSession *iam.IAM

	// S3
	s3Enabled, _ := cmd.Flags().GetBool("enable-s3")
	if s3Enabled {
		currentS3Session = s3.New(currentSession)
	}

	// IAM
	iamEnabled, _ := cmd.Flags().GetBool("enable-iam")
	if iamEnabled {
		currentIAMSession = iam.New(currentSession)
	}

	for {
		// check s3
		if s3Enabled {
			logrus.Debugf("Listing all S3 buckets in region %s.", *currentS3Session.Config.Region)
			DeleteExpiredBuckets(*currentS3Session, tagName, dryRun)
		}

		// check IAM
		if iamEnabled {
			logrus.Debug("Listing all IAM access.")
			iam2.DeleteExpiredIAM(currentIAMSession, tagName, dryRun)
		}

		time.Sleep(time.Duration(interval) * time.Second)
	}
}