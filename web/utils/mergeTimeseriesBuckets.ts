/** Одна колонка графика: один или несколько исходных бакетов. */
export type TimeseriesBucketGroup = {
  keys: string[];
  sortKeyStart: string;
  sortKeyEnd: string;
};

const MIN_BAR_PX = 16;

/** Сколько столбцов помещается по ширине контейнера. */
export function maxTimeseriesBarsForWidth(containerWidthPx: number): number {
  if (containerWidthPx <= 0) {
    return 24;
  }
  return Math.max(6, Math.floor(containerWidthPx / MIN_BAR_PX));
}

/** Объединяет соседние бакеты, если их больше, чем maxBars. */
export function groupTimeseriesBucketKeys(
  bucketKeys: string[],
  maxBars: number,
): TimeseriesBucketGroup[] {
  if (bucketKeys.length === 0) {
    return [];
  }
  if (bucketKeys.length <= maxBars) {
    return bucketKeys.map((k) => ({
      keys: [k],
      sortKeyStart: k,
      sortKeyEnd: k,
    }));
  }
  const groupSize = Math.ceil(bucketKeys.length / maxBars);
  const groups: TimeseriesBucketGroup[] = [];
  for (let i = 0; i < bucketKeys.length; i += groupSize) {
    const slice = bucketKeys.slice(i, i + groupSize);
    groups.push({
      keys: slice,
      sortKeyStart: slice[0]!,
      sortKeyEnd: slice[slice.length - 1]!,
    });
  }
  return groups;
}

export function sumSeriesInBucketGroup(
  keys: string[],
  seriesLabel: string,
  valueByKeyAndSeries: Map<string, number>,
): number {
  let sum = 0;
  for (const key of keys) {
    sum += valueByKeyAndSeries.get(`${key}\0${seriesLabel}`) ?? 0;
  }
  return sum;
}

export function buildValueLookup(
  rows: { sortKey: string; series: string; value: number }[],
): Map<string, number> {
  const m = new Map<string, number>();
  for (const r of rows) {
    m.set(`${r.sortKey}\0${r.series}`, r.value);
  }
  return m;
}
