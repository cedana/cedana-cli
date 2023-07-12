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
from asvm_part import asvm_SVC

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

def text_create(path, name, msg):
    full_path = path + "/" + name + '.txt'
    file = open(full_path, 'w')
    file.write(str(msg))
    
    
if __name__=='__main__':
    mp.set_start_method('spawn')
    dic={'brain':(300,2000)}
    
    for organ in dic:
        run_time=None
        value_filter="tissue_general=='"+organ+"' and assay!='Smart-seq2'"
        model_name='SVC'
        num_features = dic[organ][0]
        num_samples=dic[organ][1]
        init_samples=dic[organ][1]
        max_iter=-1
        for balance in [True, False]:
            if balance:
                class_weight='balanced'
            else:
                class_weight=None
            folder='./results_remove/'+value_filter.split('\'')[1]
            if run_time is None:
                path=folder+'/'+model_name+'_'+str(balance)+'_'+str(num_features)+'_'+str(num_samples)
            else:
                path=folder+'/'+model_name+'_'+str(balance)+'_'+str(num_features)+'_'+str(num_samples)+'_'+str(run_time)+'t'
            try:
                os.mkdir('results_remove')
            except OSError:
                print ("Creation of the directory %s failed" % 'results_remove')
            else:
                print ("Successfully created the directory %s " % 'results_remove')
            try:
                os.mkdir(folder)
            except OSError:
                print ("Creation of the directory %s failed" % folder)
            else:
                print ("Successfully created the directory %s " % folder)
            try:
                os.mkdir(path)
            except OSError:
                print ("Creation of the directory %s failed" % path)
            else:
                print ("Successfully created the directory %s " % path)

            with cellxgene_census.open_soma() as census:
                adata=cellxgene_census.get_anndata(census, "Homo sapiens", obs_value_filter=value_filter, 
                                                      column_names={"obs": ['soma_joinid',"cell_type"]})
                human = census["census_data"]["homo_sapiens"]
                gene_names=human.ms["RNA"].var.read().concat().to_pandas().feature_name.values

            tmp,_=scanpy.pp.filter_genes(adata,min_counts=1,inplace=False)
            gene_filtered=[j for j,x in enumerate(tmp) if x and gene_names[j][:2]!='RP' and gene_names[j][:3] not in ['HLA','MT-','MIR'] and gene_names[j][:4]!='LINC' and gene_names[j]!='MALAT1']
            df=pd.DataFrame({'gene_filtered':gene_filtered})
            df.to_csv(folder+'/gene_filtered.csv')
                
            with cellxgene_census.open_soma() as census:
                obs_df = census["census_data"]["homo_sapiens"].obs.read(value_filter = value_filter, column_names = 
                        ['soma_joinid',"cell_type"]).concat().to_pandas().set_index('soma_joinid')
            tmp=obs_df.cell_type.value_counts()
            cell_filtered=obs_df[obs_df.cell_type.isin(tmp[tmp>=10].index.values)].index.values
            df=pd.DataFrame({'cell_filtered':cell_filtered})
            df.to_csv(folder+'/cell_filtered.csv')
            
            with cellxgene_census.open_soma() as census:
                human = census["census_data"]["homo_sapiens"]
                obs_df = human.obs.read(value_filter = value_filter, column_names = ['soma_joinid',"cell_type"]).concat().to_pandas().set_index('soma_joinid').filter(items = cell_filtered, axis=0)
                var_df=human.ms["RNA"].var.read().concat().to_pandas().filter(items = gene_filtered, axis=0).reset_index()
                cell_type = obs_df.cell_type.unique()
                print(obs_df.cell_type.value_counts())
                census.close()
            t=Timer()
            t.start()
            if model_name=='SVC':
                feature_selected, num_samples_list, train_errors,test_errors,train_scores,test_scores,test_soma= asvm_SVC(path,
                value_filter=value_filter,gene_filtered=gene_filtered,cell_filtered=cell_filtered, 
                    num_features=num_features,num_samples=num_samples,init_samples=init_samples, balance=balance,
                class_weight=class_weight, max_iter=max_iter)
            else:
                feature_selected, num_samples_list, train_errors,test_errors,train_scores,test_scores,test_soma= asvm_LinearSVC(path,
                    value_filter=value_filter,gene_filtered=gene_filtered,cell_filtered=cell_filtered, 
                    num_features=num_features,num_samples=num_samples,init_samples=init_samples, balance=balance,
                    class_weight=class_weight, max_iter=max_iter,loss='hinge',tol=1e-3)

            elapsed_time=t.stop()

            text_create(path,'elapsed_time',str(elapsed_time))

            plt.figure(figsize=(8,8))
            plt.plot(train_scores,linewidth=2)
            plt.plot(test_scores,linewidth=2)
            plt.legend(['train acc','test acc'],prop = {'size':18})
            plt.xlabel('number of genes',fontdict={'weight':'normal','size': 18})
            plt.ylabel('accuracy',fontdict={'weight':'normal','size': 18})
            plt.tick_params(labelsize=18)
            plt.savefig(path+'/acc.pdf', bbox_inches="tight")

            plt.figure(figsize=(8,5))
            plt.plot(num_samples_list,linewidth=2)
            plt.xlabel('number of genes',fontdict={'weight':'normal','size': 18})
            plt.ylabel('number of cells acquired',fontdict={'weight':'normal','size': 18})
            plt.tick_params(labelsize=18)
            plt.savefig(path+'/cells.pdf', bbox_inches="tight")

