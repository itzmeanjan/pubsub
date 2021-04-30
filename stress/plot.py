#!/usr/bin/python3

from matplotlib import pyplot as plt
import seaborn as sns
from itertools import chain


def spsc(file: str):

    with open(file, 'r') as f:
        x = []
        y = []
        for line in f.readlines():
            (_x, _y) = line.strip().split(',')
            x.append(_x.strip())
            y.append(int(_y.strip())/1000)

        with plt.style.context('seaborn-bright'):
            fig = plt.Figure(figsize=(16, 9), dpi=100)

            sns.barplot(x=x, y=y, orient='v', ax=fig.gca(), color='black')

            for j, k in enumerate(fig.gca().patches):
                fig.gca().text(k.get_x() + k.get_width() / 2,
                               k.get_y() + k.get_height() * 1.01,
                               f'{int(y[j]/10**3)} ms',
                               ha='center',
                               fontsize=11,
                               color='black')

            fig.gca().set_xlabel('Data consumed by each Consumer', labelpad=12)
            fig.gca().set_ylabel('Time', labelpad=12)
            fig.gca().set_title('Single Producer Single Consumer', pad=16, fontsize=20)

            fig.savefig('spsc.png', bbox_inches='tight', pad_inches=.5)
            plt.close(fig)


def spmc(file):

    with open(file, 'r') as f:
        x = []
        hue = []
        y = []
        for line in f.readlines():
            (_x, _hue, _y) = line.strip().split(',')
            x.append(_x.strip())
            hue.append(int(_hue.strip()))
            y.append(int(_y.strip())/1000)

        hue = list(chain.from_iterable(
            [list(map(lambda c: f'{c} Consumer(s)', hue[i:i+3])) for i in range(0, len(hue), 3)]))
        label = list(chain.from_iterable(
            zip(*[y[i:i+3] for i in range(0, len(y), 3)])))

        with plt.style.context('seaborn-bright'):
            fig = plt.Figure(figsize=(16, 9), dpi=100)

            sns.barplot(x=x, y=y, hue=hue, orient='v',
                        ax=fig.gca(), palette="Blues_d")

            for j, k in enumerate(fig.gca().patches):
                fig.gca().text(k.get_x() + k.get_width() / 2,
                               k.get_y() + k.get_height() * 1.01,
                               int(label[j]/10**3),
                               ha='center',
                               fontsize=11,
                               color='k')

            fig.gca().set_xlabel('Data consumed by each Consumer', labelpad=12)
            fig.gca().set_ylabel('Time in Milliseconds', labelpad=12)
            fig.gca().set_title('Single Producer Multiple Consumers', pad=16, fontsize=20)

            fig.savefig('spmc.png', bbox_inches='tight', pad_inches=.5)
            plt.close(fig)


def main():
    try:
        spsc('spsc.csv')
        spmc('spmc.csv')
    except Exception as e:
        print(f'Error : {e}')


if __name__ == '__main__':
    main()
