/*
Copyright 2025.

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

package controller

import (
	"context"
	"fmt"
	extpg "github.com/boraxhacker/external-postgres/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"time"
)

// PostgresDatabaseReconciler reconciles a PostgresDatabase object
type PostgresDatabaseReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=external-postgres.boraxhacker,resources=postgresdatabases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=external-postgres.boraxhacker,resources=postgresdatabases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=external-postgres.boraxhacker,resources=postgresdatabases/finalizers,verbs=update
// +kubebuilder:rbac:groups=external-postgres.boraxhacker,resources=postgresinstances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=external-postgres.boraxhacker,resources=postgresinstances/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=external-postgres.boraxhacker,resources=postgresinstances/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=secrets;configmaps,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.4/pkg/reconcile
func (r *PostgresDatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.Log.WithValues("database", req.Name)

	// 1. Fetch the Database instance
	dbcr := &extpg.PostgresDatabase{}
	if err := r.Get(ctx, req.NamespacedName, dbcr); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Database resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get Database")
		return ctrl.Result{}, err
	}

	// 2. Fetch the referenced Instance
	instcr := &extpg.PostgresInstance{}
	instkey := client.ObjectKey{Name: dbcr.Spec.InstanceRef, Namespace: dbcr.Namespace}
	if err := r.Get(ctx, instkey, instcr); err != nil {
		log.Error(err, "Failed to get Instance")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	pginst, err := r.createPostgresInstance(ctx, instcr)
	if err != nil {
		log.Error(err, "Failed to retrieve values for postgres-instance")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	pgdb, err := r.createPostgresDatabase(ctx, dbcr)
	if err != nil {
		log.Error(err, "Failed to retrieve values for postgres-database")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	err = pginst.updateDatabase(ctx, pgdb)
	if err != nil {
		log.Error(err, "Failed to create/update postgres-database")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	dbcr.Status.LastSyncTime = v1.Now()
	if err := r.Status().Update(ctx, dbcr); err != nil {
		log.Error(err, "Failed to update Database status")
		return ctrl.Result{}, err
	}

	log.Info("Success create/update",
		"role", dbcr.Spec.OwnerRoleName, "database", dbcr.Spec.DatabaseName)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PostgresDatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&extpg.PostgresDatabase{}).
		Named("postgresdatabase").
		Complete(r)
}

func (r *PostgresDatabaseReconciler) createPostgresInstance(
	ctx context.Context, instcr *extpg.PostgresInstance) (*pginstance, error) {

	var err error
	var result pginstance

	result.host, err = r.retrieveValue(ctx, &instcr.Spec.Host, instcr.Namespace)
	if err != nil {
		return nil, err
	}

	portstr, err := r.retrieveValue(ctx, &instcr.Spec.Port, instcr.Namespace)
	if err != nil {
		return nil, err
	}

	result.port, err = strconv.Atoi(portstr)
	if err != nil {
		return nil, err
	}

	result.username, err = r.retrieveValue(ctx, &instcr.Spec.AdminUserName, instcr.Namespace)
	if err != nil {
		return nil, err
	}

	result.password, err = r.retrieveValue(ctx, &instcr.Spec.AdminPassword, instcr.Namespace)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *PostgresDatabaseReconciler) createPostgresDatabase(
	ctx context.Context, dbcr *extpg.PostgresDatabase) (*pgdatabase, error) {

	var err error
	var result pgdatabase

	result.name, err = r.retrieveValue(ctx, &dbcr.Spec.DatabaseName, dbcr.Namespace)
	if err != nil {
		return nil, err
	}

	result.role, err = r.retrieveValue(ctx, &dbcr.Spec.OwnerRoleName, dbcr.Namespace)
	if err != nil {
		return nil, err
	}

	result.password, err = r.retrieveValue(ctx, &dbcr.Spec.OwnerPassword, dbcr.Namespace)
	if err != nil {
		return nil, err
	}

	result.keepUpdated = dbcr.Spec.KeepUpdated

	return &result, nil
}

func (r *PostgresDatabaseReconciler) retrieveValue(
	ctx context.Context, vv *extpg.VarValue, ns string) (string, error) {

	if vv.Value != "" {

		return vv.Value, nil
	}

	if vv.ValueFrom.ConfigMapKeyRef != nil {
		return r.retrieveValueFromConfigMap(ctx, vv.ValueFrom.ConfigMapKeyRef, ns)
	}

	return r.retrieveValueFromSecret(ctx, vv.ValueFrom.SecretKeyRef, ns)
}

func (r *PostgresDatabaseReconciler) retrieveValueFromConfigMap(
	ctx context.Context, vv *extpg.VarKeySelector, ns string) (string, error) {

	cm := &corev1.ConfigMap{}
	key := client.ObjectKey{Name: vv.Name, Namespace: ns}
	if err := r.Get(ctx, key, cm); err != nil {
		return "", err
	}

	result, ok := cm.Data[vv.Key]
	if !ok {
		return "", fmt.Errorf("key %s not found in configmap %s", vv.Key, vv.Name)
	}

	return result, nil
}

func (r *PostgresDatabaseReconciler) retrieveValueFromSecret(
	ctx context.Context, vv *extpg.VarKeySelector, ns string) (string, error) {

	secret := &corev1.Secret{}
	key := client.ObjectKey{Name: vv.Name, Namespace: ns}
	if err := r.Get(ctx, key, secret); err != nil {
		return "", err
	}

	result, ok := secret.Data[vv.Key]
	if !ok {
		return "", fmt.Errorf("key %s not found in secret %s", vv.Key, vv.Name)
	}

	return string(result), nil
}
