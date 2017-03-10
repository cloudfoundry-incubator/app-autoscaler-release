package app

import (
	"acceptance/config"
	"fmt"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
	"strconv"
	"strings"
	"time"
)

var _ = Describe("AutoScaler specific date schedule policy", func() {
	var (
		appName              string
		appGUID              string
		instanceName         string
		initialInstanceCount int
		location             *time.Location
		endDateTime          time.Time
	)

	BeforeEach(func() {
		instanceName = generator.PrefixedRandomName("autoscaler", "service")
		createService := cf.Cf("create-service", cfg.ServiceName, cfg.ServicePlan, instanceName).Wait(cfg.DefaultTimeoutDuration())
		Expect(createService).To(Exit(0), "failed creating service")
	})

	JustBeforeEach(func() {
		appName = generator.PrefixedRandomName("autoscaler", "nodeapp")
		countStr := strconv.Itoa(initialInstanceCount)
		createApp := cf.Cf("push", appName, "--no-start", "-i", countStr, "-b", cfg.NodejsBuildpackName, "-m", cfg.NodeMemoryLimit, "-p", config.NODE_APP, "-d", cfg.AppsDomain).Wait(cfg.DefaultTimeoutDuration())
		Expect(createApp).To(Exit(0), "failed creating app")

		guid := cf.Cf("app", appName, "--guid").Wait(cfg.DefaultTimeout)
		Expect(guid).To(Exit(0))
		appGUID = strings.TrimSpace(string(guid.Out.Contents()))

		Expect(cf.Cf("start", appName).Wait(cfg.DefaultTimeout * 2)).To(Exit(0))
		waitForNInstancesRunning(appGUID, initialInstanceCount, cfg.DefaultTimeoutDuration())
	})

	AfterEach(func() {
		deleteService := cf.Cf("delete-service", instanceName, "-f").Wait(cfg.DefaultTimeoutDuration())
		Expect(deleteService).To(Exit(0))

		appReport(appName, cfg.DefaultTimeoutDuration())
		Expect(cf.Cf("delete", appName, "-f", "-r").Wait(cfg.CfPushTimeoutDuration())).To(Exit(0))
	})

	Context("when scale out by schedule", func() {

		JustBeforeEach(func() {
			policyByte := readPolicyFromFile("../assets/file/policy/specificdate.json")
			timeZone := "GMT"
			location, _ = time.LoadLocation(timeZone)
			timeNowInTimeZoneWithOffset := time.Now().In(location).Add(70 * time.Second)
			startDateTime := timeNowInTimeZoneWithOffset
			endDateTime = timeNowInTimeZoneWithOffset.Add(4 * time.Minute)

			policyStr := setSpecificDateScheduleDateTime(policyByte, timeZone, startDateTime, endDateTime)
			bindService := cf.Cf("bind-service", appName, instanceName, "-c", policyStr).Wait(cfg.DefaultTimeoutDuration())
			Expect(bindService).To(Exit(0), "failed binding service to app with a policy ")
		})

		AfterEach(func() {
			unbindService := cf.Cf("unbind-service", appName, instanceName).Wait(cfg.DefaultTimeoutDuration())
			Expect(unbindService).To(Exit(0), "failed unbinding service from app")
		})

		Context("with 1 instance initially", func() {
			BeforeEach(func() {
				initialInstanceCount = 1
			})

			It("should scale", func() {
				totalTime := time.Duration(cfg.ReportInterval*2)*time.Second + 2*time.Minute
				By("Start schedule")
				Eventually(func() int {
					return runningInstances(appGUID, totalTime)
				}, totalTime, 15*time.Second).Should(Equal(3))

				By("Within schedule")
				jobRunTime := endDateTime.Sub(time.Now().In(location))
				Consistently(func() int {
					return runningInstances(appGUID, jobRunTime)
				}, jobRunTime, 15*time.Second).Should(BeNumerically(">=", 2))

				By("End schedule")
				Eventually(func() int {
					return runningInstances(appGUID, totalTime)
				}, totalTime, 15*time.Second).Should(Equal(1))
			})
		})
	})

})

func setSpecificDateScheduleDateTime(policyByte []byte, timeZone string, startDateTime time.Time, endDateTime time.Time) string {
	dateTimeParseFormat := "2006-01-02T15:04"
	startDateTimeStr := startDateTime.Format(dateTimeParseFormat)
	endDateTimeStr := endDateTime.Format(dateTimeParseFormat)
	return fmt.Sprintf(string(policyByte), timeZone, startDateTimeStr, endDateTimeStr)
}
