import cellxgene_census
import numpy as np
import pandas as pd
import warnings
import os
import sys
import time
import random
import math

import matplotlib.pyplot as plt
plt.switch_backend('agg')
from matplotlib import gridspec

import multiprocessing as mp
from sklearn.svm import SVC, LinearSVC
from sklearn.metrics import mean_squared_error 
from sklearn.metrics.pairwise import cosine_similarity
from sklearn.preprocessing import LabelEncoder
import scanpy

class TimerError(Exception):
     """A custom exception used to report errors in use of Timer class"""

class Timer:
    def __init__(self):
        self._start_time = None

    def start(self):
        if self._start_time is not None:
            raise TimerError(f"Timer is running. Use .stop() to stop it")

        self._start_time = time.perf_counter()

    def stop(self):
        if self._start_time is None:
            raise TimerError(f"Timer is not running. Use .start() to start it")

        elapsed_time = time.perf_counter() - self._start_time
        self._start_time = None
        print(f"Elapsed time: {elapsed_time:0.4f} seconds")
        return elapsed_time


def get_error(model, X, y,sample_weight):
    y_pred = model.predict(X)
    return mean_squared_error(y_pred, y,sample_weight=sample_weight)

def select_samples_mincomplexity_LinearSVC(X, y, num_samples,balance=False,penalty='l2',loss='squared_hinge',
                                           dual=True, tol=1e-4, C=1.0, fit_intercept=True,intercept_scaling=1,
                                           class_weight=None, random_state=None, max_iter=1000):
    model = LinearSVC(penalty=penalty,loss=loss,dual=dual, tol=tol, C=C, fit_intercept=fit_intercept,
                          intercept_scaling=intercept_scaling, class_weight=class_weight, 
                          random_state=random_state, max_iter=max_iter)
    model.fit(X,y)
    y_pred = model.predict(X)
    sv = [i for i in range(len(y)) if y[i] != y_pred[i]]
    if balance:
        indices=[]
        classes=np.unique(y)
        sv_classes=[list(set(list(np.where(y == c)[0])) & set(sv)) for c in classes]
        sv_classes.sort(key=len)
        for i in range(len(classes)):
            sv_class=sv_classes[i]
            at_least=int((num_samples-len(indices))/(len(classes)-i))
            if len(sv_class)<=at_least:
                indices+=sv_class
            else:
                indices += random.sample(sv_class, at_least)
    else:
        indices=sv if len(sv)<num_samples else random.sample(sv, num_samples)
    return indices, model

def select_samples_mincomplexity_SVC(X, y, num_samples,balance=False,class_weight=None, C=1.0, tol=1e-3, max_iter=1000, 
                                     cache_size=200, decision_function_shape='ovr', shrinking=True, probability=False,
                                     break_ties=False, random_state=None):
    model = SVC(kernel='linear',tol=tol, C=C, class_weight=class_weight,random_state=random_state, 
                    max_iter=max_iter, cache_size=cache_size, decision_function_shape=decision_function_shape,
                   shrinking=shrinking, probability=probability, break_ties=break_ties)
    model.fit(X, y)
    y_pred = model.predict(X)
    sv = [i for i in range(len(y)) if y[i] != y_pred[i]]
    if balance:
        indices=[]
        classes=np.unique(y)
        sv_classes=[list(set(list(np.where(y == c)[0])) & set(sv)) for c in classes]
        sv_classes.sort(key=len)
        for i in range(len(classes)):
            sv_class=sv_classes[i]
            at_least=int((num_samples-len(indices))/(len(classes)-i))
            if len(sv_class)<=at_least:
                indices+=sv_class
            else:
                indices += random.sample(sv_class, at_least)
    else:
        indices=sv if len(sv)<num_samples else random.sample(sv, num_samples)
    return indices, model


def get_angles_LinearSVC(i, X, y, feature_list,w_padded,penalty='l2',loss='squared_hinge',dual=True, tol=1e-4, C=1.0,
                         fit_intercept=True,intercept_scaling=1, class_weight=None, random_state=None, max_iter=1000):
    if not np.any(X[:,i]):
        return 0
    model = LinearSVC(penalty=penalty,loss=loss,dual=dual, tol=tol, C=C, fit_intercept=fit_intercept,
                          intercept_scaling=intercept_scaling, class_weight=class_weight, 
                          random_state=random_state, max_iter=max_iter)
    model.fit(X[:, feature_list + [i]], y)
    w_new = model.coef_
    cos=cosine_similarity(w_padded,w_new)
    angle=sum([np.emath.arccos(cos[j,j]) for j in range(w_padded.shape[0])])
    return angle

def get_angles_SVC(i, X, y, feature_list,w_padded,class_weight=None, C=1.0, tol=1e-3, max_iter=1000, 
                                     cache_size=200, decision_function_shape='ovr', shrinking=True, probability=False,
                                     break_ties=False, random_state=None):
    if not np.any(X[:,i]):
        return 0
    model=SVC(kernel='linear',tol=tol, C=C, class_weight=class_weight,random_state=random_state, 
                    max_iter=max_iter, cache_size=cache_size, decision_function_shape=decision_function_shape,
                   shrinking=shrinking, probability=probability, break_ties=break_ties)
    model.fit(X[:, feature_list + [i]], y)
    w_new = model.coef_
    cos=cosine_similarity(w_padded,w_new)
    angle=sum([np.emath.arccos(cos[j,j]) for j in range(w_padded.shape[0])])
    return angle
    
def select_feature_LinearSVC(X, y, feature_list,penalty='l2',loss='squared_hinge',dual=True, tol=1e-4, C=1.0,
                             fit_intercept=True,intercept_scaling=1, class_weight=None, random_state=None, max_iter=1000):
    model = LinearSVC(penalty=penalty,loss=loss,dual=dual, tol=tol, C=C, fit_intercept=fit_intercept,
                          intercept_scaling=intercept_scaling, class_weight=class_weight, 
                          random_state=random_state, max_iter=max_iter)
    model.fit(X[:, feature_list], y)
    coef_ = model.coef_
    w_padded = np.hstack((coef_, np.zeros((coef_.shape[0],1))))
    indices=list(set(range(X.shape[1]))-set(feature_list))
    pool = mp.Pool(mp.cpu_count())
    angles=pool.starmap(get_angles_LinearSVC, [(i, X, y,feature_list,w_padded,penalty,loss,dual,tol, C,fit_intercept,
                                                intercept_scaling,class_weight, random_state, max_iter) for i in indices])
    pool.close()
    return indices[angles.index(max(angles))]

def select_feature_SVC(X, y, feature_list,class_weight=None, C=1.0, tol=1e-3, max_iter=1000,cache_size=200,
                       decision_function_shape='ovr', shrinking=True, probability=False,break_ties=False, random_state=None):
    model=SVC(kernel='linear',tol=tol, C=C, class_weight=class_weight,random_state=random_state, 
                    max_iter=max_iter, cache_size=cache_size, decision_function_shape=decision_function_shape,
                   shrinking=shrinking, probability=probability, break_ties=break_ties)
    model.fit(X[:, feature_list], y)
    coef_ = model.coef_
    w_padded = np.hstack((coef_, np.zeros((coef_.shape[0],1))))
    indices=list(set(range(X.shape[1]))-set(feature_list))
    pool = mp.Pool(mp.cpu_count())
    angles=pool.starmap(get_angles_SVC, [(i, X, y,feature_list,w_padded,class_weight, C, tol, max_iter, cache_size, 
                                          decision_function_shape, shrinking, probability, break_ties, random_state) 
                                         for i in indices])
    pool.close()
    return indices[angles.index(max(angles))]

def get_scores_LinearSVC(i, X_global, y_global,penalty='l2',loss='squared_hinge',dual=True, tol=1e-4, C=1.0, fit_intercept=True,
                          intercept_scaling=1, class_weight=None, random_state=None, max_iter=1000):
    model = LinearSVC(penalty=penalty,loss=loss,dual=dual, tol=tol, C=C, fit_intercept=fit_intercept,
                          intercept_scaling=intercept_scaling, class_weight=class_weight, 
                          random_state=random_state, max_iter=max_iter)
    model.fit(X_global[:,i].reshape(-1, 1),y_global)
    if class_weight=='balanced':
        classes, inverse, count=np.unique(y_global,return_inverse=True, return_counts=True)
        sample_weight=(y_global.shape[0]/(len(classes)*count))[inverse]
    else:
        sample_weight=None
    return model.score(X_global[:,i].reshape(-1, 1),y_global, sample_weight=sample_weight)

def get_scores_SVC(i, X_global, y_global,class_weight=None, C=1.0, tol=1e-3, max_iter=1000,cache_size=200,
                       decision_function_shape='ovr', shrinking=True, probability=False,break_ties=False, random_state=None):
    model=SVC(kernel='linear',tol=tol, C=C, class_weight=class_weight,random_state=random_state, 
                    max_iter=max_iter, cache_size=cache_size, decision_function_shape=decision_function_shape,
                   shrinking=shrinking, probability=probability, break_ties=break_ties)
    model.fit(X_global[:,i].reshape(-1, 1),y_global)
    if class_weight=='balanced':
        classes, inverse, count=np.unique(y_global,return_inverse=True, return_counts=True)
        sample_weight=(y_global.shape[0]/(len(classes)*count))[inverse]
    else:
        sample_weight=None
    return model.score(X_global[:,i].reshape(-1, 1),y_global, sample_weight=sample_weight)


def asvm_LinearSVC(value_filter, gene_filtered, cell_filtered, num_features, num_samples,init_features=1,init_samples=None, 
                          balance=False,penalty='l2',loss='squared_hinge',dual=True, tol=1e-4, C=1.0, fit_intercept=True,
                          intercept_scaling=1, class_weight=None, random_state=None, max_iter=1000):

    with cellxgene_census.open_soma() as census:
        human = census["census_data"]["homo_sapiens"]
        obs_df = human.obs.read(value_filter = value_filter, column_names = 
                                ['soma_joinid',"cell_type"]).concat().to_pandas().set_index('soma_joinid').filter(items = cell_filtered, axis=0)
        var_df=human.ms["RNA"].var.read().concat().to_pandas().filter(items = gene_filtered, axis=0).reset_index()

        _idx=np.arange(len(obs_df))
        random.shuffle(_idx)
        train_soma=obs_df.index.values[sorted(_idx[:int(len(obs_df)*4/5)])]
        test_soma=obs_df.index.values[sorted(_idx[int(len(obs_df)*4/5):])]
        label_encoder=LabelEncoder()
        label_encoder.fit(obs_df.cell_type.values)
        y_train=label_encoder.transform(obs_df.loc[train_soma].cell_type.values)
        y_test=label_encoder.transform(obs_df.loc[test_soma].cell_type.values)
        df=pd.DataFrame({'idx':_idx})
        df.to_csv(path+'/idx.csv')
        
        feature_selected = []
        feature_soma=[]
        num_samples_list = []
        train_errors = []
        test_errors = []
        train_scores = []
        test_scores = []
        step_times=[]
        if init_samples is None:
            init_samples=num_samples

        if balance:
            samples_idx=[]
            classes=np.unique(y_train)
            sample_classes=[]
            for c in classes:
                sample_class = list(np.where(y_train == c)[0])
                sample_classes.append(sample_class)
            sample_classes.sort(key=len)
            for i in range(len(classes)):
                sample_class=sample_classes[i]
                at_least=int((init_samples-len(samples_idx))/(len(classes)-i))
                if len(sample_class)<=at_least:
                    samples_idx+=sample_class
                else:
                    samples_idx += random.sample(sample_class, at_least)
            samples = train_soma[sorted(samples_idx)]
        else:
            shuffle = np.arange(len(train_soma))
            np.random.shuffle(shuffle)
            samples = train_soma[sorted(shuffle[:init_samples])]


        adata_train=cellxgene_census.get_anndata(census, "Homo sapiens", obs_value_filter=value_filter, 
                                              column_names={"obs": ['soma_joinid',"cell_type"]}, 
                                                     obs_coords=samples,var_coords=gene_filtered)
        scanpy.pp.log1p(adata_train)
        scanpy.pp.scale(adata_train)       
        samples_global=samples
        num_samples_list.append(len(samples_global))

        pool = mp.Pool(mp.cpu_count())
        scores=pool.starmap(get_scores_LinearSVC, [(i,adata_train.X, label_encoder.transform(adata_train.obs['cell_type']),
                                          penalty,loss,dual, tol, C, fit_intercept,intercept_scaling, class_weight,
                                          random_state, max_iter) for i in range(len(gene_filtered))])
        pool.close()

        new_feature = scores.index(max(scores))
        feature_selected.append(new_feature)
        feature_soma.append(gene_filtered[new_feature])

        if class_weight=='balanced':
            classes, inverse, count=np.unique(y_train,return_inverse=True, return_counts=True)
            train_sample_weight=(y_train.shape[0]/(len(classes)*count))[inverse]
            classes, inverse, count=np.unique(y_test,return_inverse=True, return_counts=True)
            test_sample_weight=(y_test.shape[0]/(len(classes)*count))[inverse]
        else:
            train_sample_weight=None
            test_sample_weight=None

        for i in range(num_features - 1):
            adata_train=cellxgene_census.get_anndata(census, "Homo sapiens", obs_value_filter=value_filter, 
                                              column_names={"obs": ['soma_joinid',"cell_type"]}, 
                                                         obs_coords=train_soma, var_coords=feature_soma)
            scanpy.pp.log1p(adata_train)
            scanpy.pp.scale(adata_train)

            adata_test=cellxgene_census.get_anndata(census, "Homo sapiens", obs_value_filter=value_filter, 
                                              column_names={"obs": ['soma_joinid']}, 
                                                        obs_coords=test_soma, var_coords=feature_soma)
            scanpy.pp.log1p(adata_test)
            scanpy.pp.scale(adata_test)

            if i==0:
                samples_idx, model = select_samples_mincomplexity_LinearSVC(adata_train.X.reshape(-1,1), y_train,
                                                                            num_samples,balance=balance,penalty=penalty,
                                                                            loss=loss,dual=dual, tol=tol, C=C,
                                                                            fit_intercept=fit_intercept,
                                                                            intercept_scaling=intercept_scaling, 
                                                                            class_weight=class_weight,
                                                                            random_state=random_state, max_iter=max_iter)
            else:
                samples_idx, model = select_samples_mincomplexity_LinearSVC(adata_train.X, y_train,
                                                                            num_samples,balance=balance,penalty=penalty,
                                                                            loss=loss,dual=dual, tol=tol, C=C,
                                                                            fit_intercept=fit_intercept,
                                                                            intercept_scaling=intercept_scaling, 
                                                                            class_weight=class_weight,
                                                                            random_state=random_state, max_iter=max_iter)
            samples=train_soma[samples_idx]

            train_error = get_error(model, adata_train.X, y_train,sample_weight=train_sample_weight)
            test_error = get_error(model, adata_test.X, y_test,sample_weight=test_sample_weight)
            train_score = model.score(adata_train.X, y_train,sample_weight=train_sample_weight)
            test_score = model.score(adata_test.X, y_test,sample_weight=test_sample_weight)
            train_errors.append(train_error)
            test_errors.append(test_error)
            train_scores.append(train_score)
            test_scores.append(test_score)
            df=var_df.filter(items=feature_selected,axis=0).reset_index()
            df['training_accuracy'],df['test_accuracy'],df['training_error'],df['test_error'],df['num_cells_total']=train_scores,test_scores,train_errors,test_errors,num_samples_list
            df.to_csv(path+'/gene_selected.csv')
            print("feature " + str(i) + ' : gene ' + str(new_feature)+'  '+str(len(samples_global)) + ' samples')
            print('training error=' + str(train_error) + ' test error=' + str(test_error))
            print('training accuracy=' + str(train_score) + ' test accuracy=' + str(test_score))

            samples_global = list(set().union(samples_global, samples))
            num_samples_list.append(len(samples_global))

            adata_train=cellxgene_census.get_anndata(census, "Homo sapiens", obs_value_filter=value_filter, 
                                              column_names={"obs": ['soma_joinid',"cell_type"]}, 
                                                         obs_coords=samples,var_coords=gene_filtered)
            scanpy.pp.log1p(adata_train)
            scanpy.pp.scale(adata_train)

            new_feature=select_feature_LinearSVC(adata_train.X, label_encoder.transform(adata_train.obs['cell_type']),
                                                 feature_selected,penalty=penalty,loss=loss,dual=dual, tol=tol, C=C,
                                                 fit_intercept=fit_intercept,intercept_scaling=intercept_scaling,
                                                 class_weight=class_weight,random_state=random_state, max_iter=max_iter)
            feature_selected.append(new_feature)
            feature_soma.append(gene_filtered[new_feature])

        adata_train=cellxgene_census.get_anndata(census, "Homo sapiens", obs_value_filter=value_filter, 
                                          column_names={"obs": ['soma_joinid',"cell_type"]}, 
                                                     obs_coords=train_soma, var_coords=feature_soma)
        scanpy.pp.log1p(adata_train)
        scanpy.pp.scale(adata_train)

        adata_test=cellxgene_census.get_anndata(census, "Homo sapiens", obs_value_filter=value_filter, 
                                          column_names={"obs": ['soma_joinid']}, 
                                                    obs_coords=test_soma, var_coords=feature_soma)
        scanpy.pp.log1p(adata_test)
        scanpy.pp.scale(adata_test)

        model=LinearSVC(penalty=penalty,loss=loss,dual=dual, tol=tol, C=C, fit_intercept=fit_intercept,
                              intercept_scaling=intercept_scaling, class_weight=class_weight,
                              random_state=random_state, max_iter=max_iter)
        model.fit(adata_train.X, y_train)

        train_error = get_error(model, adata_train.X, y_train,sample_weight=train_sample_weight)
        test_error = get_error(model, adata_test.X, y_test,sample_weight=test_sample_weight)
        train_score = model.score(adata_train.X, y_train,sample_weight=train_sample_weight)
        test_score = model.score(adata_test.X, y_test,sample_weight=test_sample_weight)
        train_errors.append(train_error)
        test_errors.append(test_error)
        train_scores.append(train_score)
        test_scores.append(test_score)
        df=var_df.filter(items=feature_selected,axis=0).reset_index()
        df['training_accuracy'],df['test_accuracy'],df['training_error'],df['test_error'],df['num_cells_total']=train_scores,test_scores,train_errors,test_errors,num_samples_list
        df.to_csv(path+'/gene_selected.csv')
        print("feature " + str(num_features-1) + ' : gene ' + str(new_feature)+'  '+str(len(samples_global)) + ' samples')
        print('training error=' + str(train_error) + ' test error=' + str(test_error))
        print('training accuracy=' + str(train_score) + ' test accuracy=' + str(test_score))
    
    return feature_selected, num_samples_list, train_errors, test_errors, train_scores, test_scores,test_soma


def asvm_SVC(path, value_filter, gene_filtered, cell_filtered, num_features, num_samples,init_features=1,init_samples=None, 
                          balance=False,class_weight=None, C=1.0, tol=1e-3, max_iter=1000,cache_size=200,
                       decision_function_shape='ovr', shrinking=True, probability=False,break_ties=False, random_state=None):

    with cellxgene_census.open_soma() as census:
        human = census["census_data"]["homo_sapiens"]
        obs_df = human.obs.read(value_filter = value_filter, column_names = 
                                ['soma_joinid',"cell_type"]).concat().to_pandas().set_index('soma_joinid').filter(items = cell_filtered, axis=0)
        var_df=human.ms["RNA"].var.read().concat().to_pandas().filter(items = gene_filtered, axis=0).reset_index()

        _idx=np.arange(len(obs_df))
        random.shuffle(_idx)
        train_soma=obs_df.index.values[sorted(_idx[:int(len(obs_df)*4/5)])]
        test_soma=obs_df.index.values[sorted(_idx[int(len(obs_df)*4/5):])]
        label_encoder=LabelEncoder()
        label_encoder.fit(obs_df.cell_type.values)
        y_train=label_encoder.transform(obs_df.loc[train_soma].cell_type.values)
        y_test=label_encoder.transform(obs_df.loc[test_soma].cell_type.values)
        df=pd.DataFrame({'idx':_idx})
        df.to_csv(path+'/idx.csv')

        feature_selected = []
        feature_soma=[]
        num_samples_list = []
        train_errors = []
        test_errors = []
        train_scores = []
        test_scores = []
        step_times=[]
        if init_samples is None:
            init_samples=num_samples

        if balance:
            samples_idx=[]
            classes=np.unique(y_train)
            sample_classes=[]
            for c in classes:
                sample_class = list(np.where(y_train == c)[0])
                sample_classes.append(sample_class)
            sample_classes.sort(key=len)
            for i in range(len(classes)):
                sample_class=sample_classes[i]
                at_least=int((init_samples-len(samples_idx))/(len(classes)-i))
                if len(sample_class)<=at_least:
                    samples_idx+=sample_class
                else:
                    samples_idx += random.sample(sample_class, at_least)
            samples = train_soma[sorted(samples_idx)]
        else:
            shuffle = np.arange(len(train_soma))
            np.random.shuffle(shuffle)
            samples = train_soma[sorted(shuffle[:init_samples])]


        adata_train=cellxgene_census.get_anndata(census, "Homo sapiens", obs_value_filter=value_filter, 
                                              column_names={"obs": ['soma_joinid',"cell_type"]}, 
                                                     obs_coords=samples,var_coords=gene_filtered)
        scanpy.pp.log1p(adata_train)
        scanpy.pp.scale(adata_train)       
        samples_global=samples
        num_samples_list.append(len(samples_global))

        pool = mp.Pool(mp.cpu_count())
        scores=pool.starmap(get_scores_SVC, [(i,adata_train.X, label_encoder.transform(adata_train.obs['cell_type']),
                                          class_weight, C, tol, max_iter, cache_size,decision_function_shape, shrinking,
                                              probability, break_ties, random_state) for i in range(len(gene_filtered))])
        pool.close()

        new_feature = scores.index(max(scores))
        feature_selected.append(new_feature)
        feature_soma.append(gene_filtered[new_feature])

        if class_weight=='balanced':
            classes, inverse, count=np.unique(y_train,return_inverse=True, return_counts=True)
            train_sample_weight=(y_train.shape[0]/(len(classes)*count))[inverse]
            classes, inverse, count=np.unique(y_test,return_inverse=True, return_counts=True)
            test_sample_weight=(y_test.shape[0]/(len(classes)*count))[inverse]
        else:
            train_sample_weight=None
            test_sample_weight=None

        for i in range(num_features - 1):
            adata_train=cellxgene_census.get_anndata(census, "Homo sapiens", obs_value_filter=value_filter, 
                                              column_names={"obs": ['soma_joinid',"cell_type"]}, 
                                                         obs_coords=train_soma, var_coords=feature_soma)
            scanpy.pp.log1p(adata_train)
            scanpy.pp.scale(adata_train)

            adata_test=cellxgene_census.get_anndata(census, "Homo sapiens", obs_value_filter=value_filter, 
                                              column_names={"obs": ['soma_joinid']}, 
                                                        obs_coords=test_soma, var_coords=feature_soma)
            scanpy.pp.log1p(adata_test)
            scanpy.pp.scale(adata_test)

            if i==0:
                samples_idx, model = select_samples_mincomplexity_SVC(adata_train.X.reshape(-1,1), y_train,
                                                                      num_samples,balance=balance,
                                                          tol=tol, C=C, class_weight=class_weight,
                                              random_state=random_state,max_iter=max_iter, cache_size=cache_size,
                                              decision_function_shape=decision_function_shape,shrinking=shrinking,
                                              probability=probability, break_ties=break_ties)
            else:
                samples_idx, model = select_samples_mincomplexity_SVC(adata_train.X, y_train, num_samples,balance=balance,
                                                          tol=tol, C=C, class_weight=class_weight,
                                              random_state=random_state,max_iter=max_iter, cache_size=cache_size,
                                              decision_function_shape=decision_function_shape,shrinking=shrinking,
                                              probability=probability, break_ties=break_ties)
            samples=train_soma[samples_idx]

            train_error = get_error(model, adata_train.X, y_train,sample_weight=train_sample_weight)
            test_error = get_error(model, adata_test.X, y_test,sample_weight=test_sample_weight)
            train_score = model.score(adata_train.X, y_train,sample_weight=train_sample_weight)
            test_score = model.score(adata_test.X, y_test,sample_weight=test_sample_weight)
            train_errors.append(train_error)
            test_errors.append(test_error)
            train_scores.append(train_score)
            test_scores.append(test_score)
            
            df=var_df.filter(items=feature_selected,axis=0).reset_index()
            df['training_accuracy'],df['test_accuracy'],df['training_error'],df['test_error'],df['num_cells_total']=train_scores,test_scores,train_errors,test_errors,num_samples_list
            df.to_csv(path+'/gene_selected.csv')
            print("feature " + str(i) + ' : gene ' + str(new_feature)+'  '+str(len(samples_global)) + ' samples')
            print('training error=' + str(train_error) + ' test error=' + str(test_error))
            print('training accuracy=' + str(train_score) + ' test accuracy=' + str(test_score))

            samples_global = list(set().union(samples_global, samples))
            num_samples_list.append(len(samples_global))

            adata_train=cellxgene_census.get_anndata(census, "Homo sapiens", obs_value_filter=value_filter, 
                                              column_names={"obs": ['soma_joinid',"cell_type"]}, 
                                                         obs_coords=samples,var_coords=gene_filtered)
            scanpy.pp.log1p(adata_train)
            scanpy.pp.scale(adata_train)
            
            y=label_encoder.transform(adata_train.obs['cell_type'])
            if np.unique(y).shape[0]==1:
                return feature_selected, num_samples_list[:-1], train_errors, test_errors, train_scores, test_scores,test_soma
            new_feature=select_feature_SVC(adata_train.X, y, feature_selected,
                                        tol=tol, C=C, class_weight=class_weight,
                                              random_state=random_state,max_iter=max_iter, cache_size=cache_size,
                                              decision_function_shape=decision_function_shape,shrinking=shrinking,
                                              probability=probability, break_ties=break_ties)
            feature_selected.append(new_feature)
            feature_soma.append(gene_filtered[new_feature])

        adata_train=cellxgene_census.get_anndata(census, "Homo sapiens", obs_value_filter=value_filter, 
                                          column_names={"obs": ['soma_joinid',"cell_type"]}, 
                                                     obs_coords=train_soma, var_coords=feature_soma)
        scanpy.pp.log1p(adata_train)
        scanpy.pp.scale(adata_train)

        adata_test=cellxgene_census.get_anndata(census, "Homo sapiens", obs_value_filter=value_filter, 
                                          column_names={"obs": ['soma_joinid']}, 
                                                    obs_coords=test_soma, var_coords=feature_soma)
        scanpy.pp.log1p(adata_test)
        scanpy.pp.scale(adata_test)

        model=SVC(kernel='linear', tol=tol, C=C, class_weight=class_weight,
                                              random_state=random_state,max_iter=max_iter, cache_size=cache_size,
                                              decision_function_shape=decision_function_shape,shrinking=shrinking,
                                              probability=probability, break_ties=break_ties)
        model.fit(adata_train.X, y_train)

        train_error = get_error(model, adata_train.X, y_train,sample_weight=train_sample_weight)
        test_error = get_error(model, adata_test.X, y_test,sample_weight=test_sample_weight)
        train_score = model.score(adata_train.X, y_train,sample_weight=train_sample_weight)
        test_score = model.score(adata_test.X, y_test,sample_weight=test_sample_weight)
        train_errors.append(train_error)
        test_errors.append(test_error)
        train_scores.append(train_score)
        test_scores.append(test_score)
        df=var_df.filter(items=feature_selected,axis=0).reset_index()
        df['training_accuracy'],df['test_accuracy'],df['training_error'],df['test_error'],df['num_cells_total']=train_scores,test_scores,train_errors,test_errors,num_samples_list
        df.to_csv(path+'/gene_selected.csv')
        print("feature " + str(num_features-1) + ' : gene ' + str(new_feature)+'  '+str(len(samples_global)) + ' samples')
        print('training error=' + str(train_error) + ' test error=' + str(test_error))
        print('training accuracy=' + str(train_score) + ' test accuracy=' + str(test_score))

    return feature_selected, num_samples_list, train_errors, test_errors, train_scores, test_scores,test_soma

