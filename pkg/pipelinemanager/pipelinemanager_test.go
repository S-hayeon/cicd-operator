/*
 Copyright 2021 The CI/CD Operator Authors

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package pipelinemanager

import (
	"github.com/bmizerany/assert"
	"github.com/stretchr/testify/require"
	tektonv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/apis/run/v1alpha1"
	cicdv1 "github.com/tmax-cloud/cicd-operator/api/v1"
	"github.com/tmax-cloud/cicd-operator/pkg/git"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/apis/duck/v1beta1"
	"testing"
	"time"
)

func TestAppendBaseShaToDescription(t *testing.T) {
	desc := "test description"
	sha := git.FakeSha

	appended := appendBaseShaToDescription(desc, sha)
	assert.Equal(t, desc, appended[:len(desc)], "Description")
	assert.Equal(t, statusDescriptionBaseSHAKey+git.FakeSha, appended[len(appended)-len(statusDescriptionBaseSHAKey+git.FakeSha):], "BaseSHA")

	desc = "description which is very longlonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglong"
	msgLen := statusDescriptionMaxLength - len(statusDescriptionBaseSHAKey) - len(git.FakeSha) - len(statusDescriptionEllipse)
	appended = appendBaseShaToDescription(desc, sha)
	assert.Equal(t, desc[:msgLen], appended[:len(desc[:msgLen])], "Description")
	assert.Equal(t, statusDescriptionBaseSHAKey+git.FakeSha, appended[len(appended)-len(statusDescriptionBaseSHAKey+git.FakeSha):], "BaseSHA")

	sha = ""
	appended = appendBaseShaToDescription(desc, sha)
	assert.Equal(t, desc[:statusDescriptionMaxLength], appended, "Description")
}

func TestParseBaseFromDescription(t *testing.T) {
	fullDesc := "Job is running... BaseSHA:2641c89aac959fb804ec6f2a4a22e129f4ac4900"
	sha := ParseBaseFromDescription(fullDesc)
	assert.Equal(t, "2641c89aac959fb804ec6f2a4a22e129f4ac4900", sha)

	fullDesc = "Job is running... BaseSHA:zzzzzzzzzzzzzzzzz"
	sha = ParseBaseFromDescription(fullDesc)
	assert.Equal(t, "", sha)
}

func TestGetJobRunStatus(t *testing.T) {
	tc := map[string]struct {
		prStatus tektonv1beta1.PipelineRunStatus
		job      *cicdv1.Job

		expectedJobStatus cicdv1.CommitStatusState
	}{
		"successTaskRun": {
			prStatus: tektonv1beta1.PipelineRunStatus{
				PipelineRunStatusFields: tektonv1beta1.PipelineRunStatusFields{
					TaskRuns: map[string]*tektonv1beta1.PipelineRunTaskRunStatus{
						"statusNil": {
							PipelineTaskName: "statusNilTask",
						},
						"notMatchName": {
							PipelineTaskName: "notMatchTask",
							Status: &tektonv1beta1.TaskRunStatus{
								TaskRunStatusFields: tektonv1beta1.TaskRunStatusFields{
									PodName: "notMatch",
								},
							},
						},
						"matchName": {
							PipelineTaskName: "matchTask",
							Status: &tektonv1beta1.TaskRunStatus{
								Status: v1beta1.Status{
									Conditions: v1beta1.Conditions{
										{
											Status:  corev1.ConditionTrue,
											Message: "Success",
										},
									},
								},
								TaskRunStatusFields: tektonv1beta1.TaskRunStatusFields{
									PodName:        "match",
									StartTime:      &metav1.Time{Time: time.Now().Add(-1 * time.Hour)},
									CompletionTime: &metav1.Time{Time: time.Now()},
								},
							},
						},
					},
				},
			},
			job: &cicdv1.Job{
				Container: corev1.Container{
					Name: "matchTask",
				},
			},
			expectedJobStatus: cicdv1.CommitStatusStateSuccess,
		},
		"failureTaskRun": {
			prStatus: tektonv1beta1.PipelineRunStatus{
				PipelineRunStatusFields: tektonv1beta1.PipelineRunStatusFields{
					TaskRuns: map[string]*tektonv1beta1.PipelineRunTaskRunStatus{
						"matchName": {
							PipelineTaskName: "matchTask",
							Status: &tektonv1beta1.TaskRunStatus{
								Status: v1beta1.Status{
									Conditions: v1beta1.Conditions{
										{
											Status:  corev1.ConditionFalse,
											Message: "Failed",
										},
									},
								},
								TaskRunStatusFields: tektonv1beta1.TaskRunStatusFields{
									PodName:        "match",
									StartTime:      &metav1.Time{Time: time.Now().Add(-1 * time.Hour)},
									CompletionTime: &metav1.Time{Time: time.Now()},
								},
							},
						},
					},
				},
			},
			job: &cicdv1.Job{
				Container: corev1.Container{
					Name: "matchTask",
				},
			},
			expectedJobStatus: cicdv1.CommitStatusStateFailure,
		},
		"pendingTaskRun": {
			prStatus: tektonv1beta1.PipelineRunStatus{
				PipelineRunStatusFields: tektonv1beta1.PipelineRunStatusFields{
					TaskRuns: map[string]*tektonv1beta1.PipelineRunTaskRunStatus{
						"matchName": {
							PipelineTaskName: "matchTask",
							Status: &tektonv1beta1.TaskRunStatus{
								TaskRunStatusFields: tektonv1beta1.TaskRunStatusFields{
									PodName:   "match",
									StartTime: &metav1.Time{Time: time.Now().Add(-1 * time.Hour)},
								},
							},
						},
					},
				},
			},
			job: &cicdv1.Job{
				Container: corev1.Container{
					Name: "matchTask",
				},
			},
			expectedJobStatus: cicdv1.CommitStatusStatePending,
		},
		"successRun": {
			prStatus: tektonv1beta1.PipelineRunStatus{
				PipelineRunStatusFields: tektonv1beta1.PipelineRunStatusFields{
					Runs: map[string]*tektonv1beta1.PipelineRunRunStatus{
						"statusNil": {
							PipelineTaskName: "statusNilRun",
						},
						"notMatchName": {
							PipelineTaskName: "notMatchRun",
							Status: &tektonv1alpha1.RunStatus{
								RunStatusFields: v1alpha1.RunStatusFields{
									StartTime:      &metav1.Time{Time: time.Now().Add(-1 * time.Hour)},
									CompletionTime: nil,
									Results:        nil,
								},
							},
						},
						"matchName": {
							PipelineTaskName: "matchRun",
							Status: &tektonv1alpha1.RunStatus{
								Status: v1.Status{
									Conditions: v1.Conditions{
										{
											Status:  corev1.ConditionTrue,
											Message: "Success",
										},
									},
								},
								RunStatusFields: v1alpha1.RunStatusFields{
									StartTime:      &metav1.Time{Time: time.Now().Add(-1 * time.Hour)},
									CompletionTime: &metav1.Time{Time: time.Now()},
								},
							},
						},
					},
				},
			},
			job: &cicdv1.Job{
				Container: corev1.Container{
					Name: "matchRun",
				},
			},
			expectedJobStatus: cicdv1.CommitStatusStateSuccess,
		},
		"failedRun": {
			prStatus: tektonv1beta1.PipelineRunStatus{
				PipelineRunStatusFields: tektonv1beta1.PipelineRunStatusFields{
					Runs: map[string]*tektonv1beta1.PipelineRunRunStatus{
						"matchName": {
							PipelineTaskName: "matchRun",
							Status: &tektonv1alpha1.RunStatus{
								Status: v1.Status{
									Conditions: v1.Conditions{
										{
											Status:  corev1.ConditionFalse,
											Message: "Failed",
										},
									},
								},
								RunStatusFields: v1alpha1.RunStatusFields{
									StartTime:      &metav1.Time{Time: time.Now().Add(-1 * time.Hour)},
									CompletionTime: &metav1.Time{Time: time.Now()},
								},
							},
						},
					},
				},
			},
			job: &cicdv1.Job{
				Container: corev1.Container{
					Name: "matchRun",
				},
			},
			expectedJobStatus: cicdv1.CommitStatusStateFailure,
		},
		"pendingRun": {
			prStatus: tektonv1beta1.PipelineRunStatus{
				PipelineRunStatusFields: tektonv1beta1.PipelineRunStatusFields{
					Runs: map[string]*tektonv1beta1.PipelineRunRunStatus{
						"matchName": {
							PipelineTaskName: "matchRun",
							Status: &tektonv1alpha1.RunStatus{
								Status: v1.Status{},
								RunStatusFields: v1alpha1.RunStatusFields{
									StartTime: &metav1.Time{Time: time.Now().Add(-1 * time.Hour)},
								},
							},
						},
					},
				},
			},
			job: &cicdv1.Job{
				Container: corev1.Container{
					Name: "matchRun",
				},
			},
			expectedJobStatus: cicdv1.CommitStatusStatePending,
		},
	}
	for name, c := range tc {
		t.Run(name, func(t *testing.T) {
			jobStatus := getJobRunStatus(c.prStatus, c.job)

			require.Equal(t, c.expectedJobStatus, jobStatus.State)
		})
	}
}
