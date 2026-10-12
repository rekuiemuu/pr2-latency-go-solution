package main

import (
	"encoding/csv"
	"fmt"
	"html"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"go-algo/reliability"
)

func main() {
	rows, err := readSamples("docs/reliability_samples.csv")
	if err != nil {
		log.Fatal(err)
	}
	groups := make(map[string][]reliability.Sample)
	for _, row := range rows {
		groups[row.Experiment] = append(groups[row.Experiment], row)
	}
	order := []string{"baseline", "loss_5", "loss_10", "loss_20", "delay_100_loss_5", "jitter_loss_10"}
	if err := os.MkdirAll("docs/graphs", 0755); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile("docs/graphs/average_attempts.svg", []byte(attemptsSVG(order, groups)), 0644); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile("docs/graphs/rto_jitter.svg", []byte(rtoSVG(groups["jitter_loss_10"])), 0644); err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile("docs/Reliability_Report.md", []byte(report(order, groups)), 0644); err != nil {
		log.Fatal(err)
	}
	fmt.Println("created docs/Reliability_Report.md and two SVG graphs")
}

func readSamples(path string) ([]reliability.Sample, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, err
	}
	var samples []reliability.Sample
	for _, r := range records[1:] {
		if len(r) != 7 {
			continue
		}
		seq, _ := strconv.ParseUint(r[1], 10, 16)
		attempts, _ := strconv.ParseUint(r[2], 10, 32)
		failed, _ := strconv.ParseBool(r[4])
		ack, _ := strconv.ParseFloat(r[5], 64)
		rto, _ := strconv.ParseFloat(r[6], 64)
		samples = append(samples, reliability.Sample{Experiment: r[0], Sequence: uint16(seq), Attempts: uint32(attempts), Failed: failed, AckMs: ack, RtoMs: rto})
	}
	return samples, nil
}

func report(order []string, groups map[string][]reliability.Sample) string {
	var b strings.Builder
	b.WriteString("# Отчёт о надёжной доставке UDP\n\n")
	b.WriteString("В каждой серии отправлено 50 надёжных команд `SHOOT`. `MOVEMENT` оставлена ненадёжной: устаревшую позицию выгоднее заменить новой, а выстрел нельзя потерять или применить дважды.\n\n")
	b.WriteString("| Серия | Ср. попыток | С первой попытки | Не доставлено | Ср. время до ACK, мс | Финальный RTO, мс |\n|---|---:|---:|---:|---:|---:|\n")
	for _, name := range order {
		s := reliability.Calculate(groups[name])
		fmt.Fprintf(&b, "| %s | %.3f | %.1f%% | %.1f%% | %.2f | %.2f |\n", name, s.AverageAttempts, s.FirstAttemptPct, s.FailedPct, s.AverageAckMs, s.FinalRtoMs)
	}
	b.WriteString("\n![Среднее число попыток](graphs/average_attempts.svg)\n\n![Изменение RTO в серии с джиттером](graphs/rto_jitter.svg)\n\n")
	b.WriteString("## Вывод\n\nПри росте потерь растёт среднее число попыток. Повторные ACK игнорируются, повторная команда на сервере подтверждается, но её игровой эффект не выполняется второй раз. Для пакетов с повторной отправкой RTT не используется при обновлении RTO (алгоритм Карна).\n")
	return b.String()
}

func attemptsSVG(order []string, groups map[string][]reliability.Sample) string {
	var b strings.Builder
	b.WriteString(svgStart("Среднее число попыток", 760, 420))
	for i, name := range order {
		value := reliability.Calculate(groups[name]).AverageAttempts
		x := 70 + i*110
		height := int(value * 180)
		y := 330 - height
		fmt.Fprintf(&b, `<rect x="%d" y="%d" width="65" height="%d" fill="#4f81bd"/><text x="%d" y="%d" text-anchor="middle">%.2f</text><text x="%d" y="355" text-anchor="middle" font-size="11">%s</text>`, x, y, height, x+32, y-7, value, x+32, html.EscapeString(name))
	}
	b.WriteString(`<line x1="50" y1="330" x2="740" y2="330" stroke="black"/></g></svg>`)
	return b.String()
}

func rtoSVG(samples []reliability.Sample) string {
	sort.Slice(samples, func(i, j int) bool { return samples[i].Sequence < samples[j].Sequence })
	var points []string
	for i, sample := range samples {
		x := 55 + i*13
		y := 350 - int(sample.RtoMs/2)
		if y < 45 {
			y = 45
		}
		points = append(points, fmt.Sprintf("%d,%d", x, y))
	}
	return svgStart("RTO по номеру команды: jitter_loss_10", 760, 420) + `<line x1="50" y1="350" x2="740" y2="350" stroke="black"/><polyline fill="none" stroke="#c0504d" stroke-width="2" points="` + strings.Join(points, " ") + `"/><text x="380" y="395" text-anchor="middle">Номер команды</text></g></svg>`
}

func svgStart(title string, width, height int) string {
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d"><rect width="100%%" height="100%%" fill="white"/><text x="%d" y="25" text-anchor="middle" font-family="sans-serif" font-size="18">%s</text><g font-family="sans-serif" font-size="12">`, width, height, width/2, html.EscapeString(title))
}
