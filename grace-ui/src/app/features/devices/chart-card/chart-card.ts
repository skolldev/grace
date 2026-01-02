import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';
import { ChartModule } from 'primeng/chart';
import { AggregateValue } from '../../../core';

@Component({
  selector: 'grc-chart-card',
  templateUrl: './chart-card.html',
  imports: [ChartModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ChartCard {
  title = input.required<string>();
  data = input.required<{ label: string; data: AggregateValue }[]>();

  chartData = computed(() => {
    const points = this.data();
    return {
      labels: points.map((p) => new Date(p.label).toLocaleTimeString()),
      datasets: [
        {
          label: 'Min baseline',
          data: points.map((p) => p.data.min),
          fill: false, // No fill for the bottom
          borderColor: 'transparent',
          pointRadius: 0,
          tension: 0.4,
        },
        {
          label: 'Range',
          data: points.map((p) => p.data.max),
          fill: '-1',
          backgroundColor: 'rgba(212, 175, 55, 0.15)',
          borderColor: 'transparent',
          pointRadius: 0,
          tension: 0.4,
        },
        {
          label: 'Average',
          data: points.map((p) => p.data.avg),
          fill: false,
          borderColor: '#D4AF37',
          borderWidth: 2,
          tension: 0.4,
          pointRadius: 3,
        },
      ],
    };
  });

  chartOptions = {
    maintainAspectRatio: false,
    aspectRatio: 0.6,
    plugins: {
      legend: {
        display: false,
      },
    },
    scales: {
      x: {
        ticks: {
          color: 'rgb(156, 163, 175)',
        },
        grid: {
          color: 'rgba(156, 163, 175, 0.1)',
        },
      },
      y: {
        ticks: {
          color: 'rgb(156, 163, 175)',
        },
        grid: {
          color: 'rgba(156, 163, 175, 0.1)',
        },
      },
    },
  };
}
