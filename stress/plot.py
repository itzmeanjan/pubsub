#!/usr/bin/python3

from matplotlib import pyplot as plt
import seaborn as sns

def main():
    try:

        with open('spsc.csv', 'r') as f:
            x = []
            y = []
            for line in f.readlines():
                (_x, _y) = line.strip().split(',')
                x.append(_x.strip())
                y.append(int(_y.strip())/1000)

            with plt.style.context('dark_background'):
                fig = plt.Figure(figsize=(16, 9), dpi=100)

                sns.barplot(x=x, y=y, orient='v', ax=fig.gca())

                for j, k in enumerate(fig.gca().patches):
                    fig.gca().text(k.get_x() + k.get_width() / 2,
                                   k.get_y() + k.get_height() + .2,
                                   int(y[j]/10**3),
                                   ha='center',
                                   fontsize=11,
                                   color='white')

                fig.gca().set_xlabel('Data Published/ Consumed', labelpad=12)
                fig.gca().set_ylabel('Time in `milliseconds`', labelpad=12)
                fig.gca().set_title('Single Producer Single Consumer', pad=16, fontsize=20)

                fig.savefig('spsc.png', bbox_inches='tight', pad_inches=.5)
                plt.close(fig)

    except Exception as e:
        print(f'Error : {e}')


if __name__ == '__main__':
    main()
