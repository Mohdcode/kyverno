package mutation

import (
	"context"
	"strings"

	json_patch "github.com/evanphx/json-patch/v5"
	"github.com/go-logr/logr"
	kyvernov1 "github.com/kyverno/kyverno/api/kyverno/v1"
	kyvernov2 "github.com/kyverno/kyverno/api/kyverno/v2"
	"github.com/kyverno/kyverno/pkg/config"
	engineapi "github.com/kyverno/kyverno/pkg/engine/api"
	enginecontext "github.com/kyverno/kyverno/pkg/engine/context"
	"github.com/kyverno/kyverno/pkg/engine/handlers"
	"github.com/kyverno/kyverno/pkg/engine/internal"
	"github.com/kyverno/kyverno/pkg/engine/mutate/patch"
	engineutils "github.com/kyverno/kyverno/pkg/engine/utils"
	"github.com/kyverno/kyverno/pkg/engine/variables"
	"github.com/kyverno/kyverno/pkg/imageverifycache"
	apiutils "github.com/kyverno/kyverno/pkg/utils/api"
	jsonutils "github.com/kyverno/kyverno/pkg/utils/json"
	"gomodules.xyz/jsonpatch/v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/cache"
)

type mutateImageHandler struct {
	configuration  config.Configuration
	rclientFactory engineapi.RegistryClientFactory
	ivCache        imageverifycache.Client
	ivm            *engineapi.ImageVerificationMetadata
	images         []apiutils.ImageInfo
}

func NewMutateImageHandler(
	policyContext engineapi.PolicyContext,
	resource unstructured.Unstructured,
	rule kyvernov1.Rule,
	configuration config.Configuration,
	rclientFactory engineapi.RegistryClientFactory,
	ivCache imageverifycache.Client,
	ivm *engineapi.ImageVerificationMetadata,
) (handlers.Handler, error) {
	if len(rule.VerifyImages) == 0 {
		return nil, nil
	}
	ruleImages, _, err := engineutils.ExtractMatchingImages(resource, policyContext.JSONContext(), rule, configuration)
	if err != nil {
		return nil, err
	}
	if len(ruleImages) == 0 {
		return nil, nil
	}
	return mutateImageHandler{
		configuration:  configuration,
		rclientFactory: rclientFactory,
		ivm:            ivm,
		ivCache:        ivCache,
		images:         ruleImages,
	}, nil
}

func (h mutateImageHandler) Process(
	ctx context.Context,
	logger logr.Logger,
	policyContext engineapi.PolicyContext,
	resource unstructured.Unstructured,
	rule kyvernov1.Rule,
	contextLoader engineapi.EngineContextLoader,
	exceptions []*kyvernov2.PolicyException,
) (unstructured.Unstructured, []engineapi.RuleResponse) {
	// check if there are policy exceptions that match the incoming resource
	matchedExceptions := engineutils.MatchesException(exceptions, policyContext, logger)
	if len(matchedExceptions) > 0 {
		exceptions := make([]engineapi.GenericException, 0, len(matchedExceptions))
		var keys []string
		for i, exception := range matchedExceptions {
			key, err := cache.MetaNamespaceKeyFunc(&matchedExceptions[i])
			if err != nil {
				logger.Error(err, "failed to compute policy exception key", "namespace", exception.GetNamespace(), "name", exception.GetName())
				return resource, handlers.WithError(rule, engineapi.Mutation, "failed to compute exception key", err)
			}
			keys = append(keys, key)
			exceptions = append(exceptions, engineapi.NewPolicyException(&exception))
		}

		logger.V(3).Info("policy rule is skipped due to policy exceptions", "exceptions", keys)
		return resource, handlers.WithResponses(
			engineapi.RuleSkip(rule.Name, engineapi.Mutation, "rule is skipped due to policy exceptions"+strings.Join(keys, ", "), rule.ReportProperties).WithExceptions(exceptions),
		)
	}

	jsonContext := policyContext.JSONContext()
	ruleCopy, err := substituteVariables(rule, jsonContext, logger)
	if err != nil {
		return resource, handlers.WithResponses(
			engineapi.RuleError(rule.Name, engineapi.ImageVerify, "failed to substitute variables", err, rule.ReportProperties),
		)
	}

	var engineResponses []*engineapi.RuleResponse
	var patches []jsonpatch.JsonPatchOperation
	for _, imageVerify := range ruleCopy.VerifyImages {
		rclient, err := h.rclientFactory.GetClient(ctx, imageVerify.ImageRegistryCredentials)
		if err != nil {
			return resource, handlers.WithResponses(
				engineapi.RuleError(rule.Name, engineapi.ImageVerify, "failed to fetch secrets", err, rule.ReportProperties),
			)
		}
		iv := internal.NewImageVerifier(logger, rclient, h.ivCache, policyContext, *ruleCopy, h.ivm)
		patch, ruleResponse := iv.Verify(ctx, imageVerify, h.images, h.configuration)
		patches = append(patches, patch...)
		engineResponses = append(engineResponses, ruleResponse...)
	}
	if len(patches) != 0 {
		patch := jsonutils.JoinPatches(patch.ConvertPatches(patches...)...)
		decoded, err := json_patch.DecodePatch(patch)
		if err != nil {
			return resource, handlers.WithResponses(
				engineapi.RuleError(rule.Name, engineapi.ImageVerify, "failed to decode patch", err, rule.ReportProperties),
			)
		}
		options := &json_patch.ApplyOptions{SupportNegativeIndices: true, AllowMissingPathOnRemove: true, EnsurePathExistsOnAdd: true}
		resourceBytes, err := resource.MarshalJSON()
		if err != nil {
			return resource, handlers.WithResponses(
				engineapi.RuleError(rule.Name, engineapi.ImageVerify, "failed to marshal resource", err, rule.ReportProperties),
			)
		}
		patchedResourceBytes, err := decoded.ApplyWithOptions(resourceBytes, options)
		if err != nil {
			return resource, handlers.WithResponses(
				engineapi.RuleError(rule.Name, engineapi.ImageVerify, "failed to apply patch", err, rule.ReportProperties),
			)
		}
		if err := resource.UnmarshalJSON(patchedResourceBytes); err != nil {
			return resource, handlers.WithResponses(
				engineapi.RuleError(rule.Name, engineapi.ImageVerify, "failed to unmarshal resource", err, rule.ReportProperties),
			)
		}
	}
	return resource, handlers.WithResponses(engineResponses...)
}

func substituteVariables(rule kyvernov1.Rule, ctx enginecontext.EvalInterface, logger logr.Logger) (*kyvernov1.Rule, error) {
	// remove attestations as variables are not substituted in them
	hasValidateImageVerification := rule.HasValidateImageVerification()
	ruleCopy := *rule.DeepCopy()
	for i := range ruleCopy.VerifyImages {
		for j := range ruleCopy.VerifyImages[i].Attestations {
			ruleCopy.VerifyImages[i].Attestations[j].Conditions = nil
		}
		if hasValidateImageVerification {
			ruleCopy.VerifyImages[i].Validation.Deny.RawAnyAllConditions = nil
		}
	}

	var err error
	ruleCopy, err = variables.SubstituteAllInRule(logger, ctx, ruleCopy)
	if err != nil {
		return nil, err
	}

	// replace attestations
	for i := range ruleCopy.VerifyImages {
		for j := range ruleCopy.VerifyImages[i].Attestations {
			ruleCopy.VerifyImages[i].Attestations[j].Conditions = rule.VerifyImages[i].Attestations[j].Conditions
		}
		if hasValidateImageVerification {
			ruleCopy.VerifyImages[i].Validation.Deny.RawAnyAllConditions = rule.VerifyImages[i].Validation.Deny.RawAnyAllConditions
		}
	}
	return &ruleCopy, nil
}
