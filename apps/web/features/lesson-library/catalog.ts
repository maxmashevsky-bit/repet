export const lessonCatalog = [
  { slug: "parity", title: "Чётность и чередование", description: "Замечаем закономерности, чередуем и доказываем.", file: "parity.html", topic: "Закономерности", mark: "2n", section: "" },
  { slug: "plus-minus-one", title: "Эффект плюс-минус один", description: "Разбираемся с промежутками, границами и подсчётом.", file: "plus-minus-one.html", topic: "Подсчёт", mark: "±1", section: "" },
  { slug: "from-the-end", title: "Анализ с конца", description: "Идём от результата к исходным условиям.", file: "from-the-end.html", topic: "Стратегии", mark: "←", section: "" },
  { slug: "euler", title: "Круги Эйлера", description: "Рисуем множества и находим их пересечения.", file: "euler-graphs.html", topic: "Наглядные методы", mark: "∩", section: "euler" },
  { slug: "graphs", title: "Графы и связи", description: "Изучаем вершины, рёбра и задачи о рукопожатиях.", file: "euler-graphs.html", topic: "Наглядные методы", mark: "↗", section: "graphs" },
  { slug: "logic-work-motion", title: "Логика, работа и движение", description: "От рассуждений — к задачам о трубах и движении.", file: "logic-work-motion.html", topic: "Практика", mark: "v·t", section: "" },
  { slug: "examples", title: "Примеры и случаи", description: "Перебираем варианты системно и ничего не упускаем.", file: "examples.html", topic: "Стратегии", mark: "{…}", section: "" },
  { slug: "word-problems", title: "Текстовые задачи", description: "Переводим условие на язык схем и вычислений.", file: "word-problems.html", topic: "Практика", mark: "x=?", section: "" },
  { slug: "logic", title: "Логика", description: "Проверяем утверждения и строим точные рассуждения.", file: "logic.html", topic: "Рассуждения", mark: "⇒", section: "" },
  { slug: "constructions", title: "Конструкции и процессы", description: "Исследуем изменения шаг за шагом на моделях.", file: "constructions.html", topic: "Закономерности", mark: "n+1", section: "" },
  { slug: "combinatorics-one", title: "Комбинаторика — 1", description: "Системно перебираем варианты и открываем правила подсчёта.", file: "combinatorics-one.html", topic: "Комбинаторика", mark: "A×B", section: "" },
  { slug: "how-many", title: "Сколько?", description: "Сравниваем количества, уравниваем и находим неизвестное.", file: "how-many.html", topic: "Подсчёт", mark: "=?", section: "" },
  { slug: "logic-two", title: "Логика — 2", description: "Развиваем рассуждения и проверяем выводы на новых задачах.", file: "logic-two.html", topic: "Рассуждения", mark: "⇒", section: "" },
  { slug: "plus-minus-two", title: "Эффект плюс-минус один — 2", description: "Исследуем дни, границы и промежутки на интерактивных моделях.", file: "plus-minus-two.html", topic: "Подсчёт", mark: "±1", section: "" },
  { slug: "grid-figures", title: "Клетчатые фигуры", description: "Строим на клетчатом поле и изучаем свойства фигур.", file: "grid-figures.html", topic: "Геометрия", mark: "▦", section: "" },
  { slug: "count-the-extra", title: "Учти лишнее", description: "Считаем через дополнение и убираем повторный учёт.", file: "count-the-extra.html", topic: "Подсчёт", mark: "A∪B", section: "" },
  { slug: "combinatorics-two", title: "Комбинаторика — 2", description: "Закрепляем приёмы подсчёта и разбираем сложные сочетания.", file: "combinatorics-two.html", topic: "Комбинаторика", mark: "n!", section: "" },
  { slug: "last-digit", title: "Последняя цифра", description: "Находим циклы последних цифр и объясняем закономерности.", file: "last-digit.html", topic: "Закономерности", mark: "…7", section: "" },
  { slug: "contradiction", title: "Метод от противного", description: "Предполагаем обратное и шаг за шагом приходим к противоречию.", file: "contradiction.html", topic: "Доказательства", mark: "≠", section: "" },
  { slug: "many-and-few", title: "Конструкции: много и мало", description: "Строим примеры и исследуем наибольшие и наименьшие количества.", file: "many-and-few.html", topic: "Конструкции", mark: "min", section: "" }
] as const;

export function findLesson(slug: string) {
  return lessonCatalog.find((lesson) => lesson.slug === slug);
}
