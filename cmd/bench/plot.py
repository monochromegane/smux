import pandas as pd
import matplotlib.pyplot as plt
from IPython import embed

names = ['proto', 'jobs', 'concurrent', 'sec', 'jobs/sec', 'error', 'error_rate', 'delay']
df = pd.read_csv('result.txt', header=None, delimiter=',', names=names)

df1 = df.loc[:,['proto', 'concurrent', 'jobs/sec']].groupby(['concurrent', 'proto']).mean()['jobs/sec'].unstack()
df2 = df.loc[:,['proto', 'concurrent', 'error_rate']].groupby(['concurrent', 'proto']).mean()['error_rate'].unstack()

fig, axes = plt.subplots(nrows=2, figsize=(8, 8), )
fig.suptitle("Benchmark of HTTP and Smux (Jobs: {:,} req, Delay: {} ms)".format(df.loc[0]['jobs'], df.loc[0]['delay']))
axes[0].set_ylabel('requests/sec')
axes[1].set_ylabel('error rate')

df1.plot(kind='line', ax=axes[0])
df2.plot(kind='bar',  ax=axes[1])

plt.savefig('out.png')
