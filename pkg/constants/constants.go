package constants

import (
	"fmt"

	"github.com/shipwright-io/build/pkg/apis/build/v1alpha1"

	tknv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tknv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

var (
	TektonAPIv1alpha1 = fmt.Sprintf(
		"%s/%s",
		tknv1alpha1.SchemeGroupVersion.Group,
		tknv1alpha1.SchemeGroupVersion.Version,
	)
	TektonAPIv1beta1 = fmt.Sprintf("%s/%s",
		tknv1beta1.SchemeGroupVersion.Group,
		tknv1beta1.SchemeGroupVersion.Version,
	)
	ShipwrightAPIVersion = fmt.Sprintf(
		"%s/%s",
		v1alpha1.SchemeGroupVersion.Group,
		v1alpha1.SchemeGroupVersion.Version,
	)
)
