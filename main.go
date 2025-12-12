package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Vacancy struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Employer struct {
		Name string `json:"name"`
	} `json:"employer"`
	AlternateURL string `json:"alternate_url"`
	Area         struct {
		Name string `json:"name"`
	} `json:"area"`
	PublishedAt string `json:"published_at"`
}

type VacanciesResponse struct {
	Items []Vacancy `json:"items"`
	Found int       `json:"found"`
	Pages int       `json:"pages"`
	Page  int       `json:"page"`
}

type VacancyDetails struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Employer struct {
		Name string `json:"name"`
	} `json:"employer"`
	AlternateURL string `json:"alternate_url"`
	Area         struct {
		Name string `json:"name"`
	} `json:"area"`
	PublishedAt string `json:"published_at"`
	Salary      *struct {
		From     *int   `json:"from"`
		To       *int   `json:"to"`
		Currency string `json:"currency"`
	} `json:"salary"`
	Contacts struct {
		Name   string `json:"name"`
		Email  string `json:"email"`
		Phones []struct {
			Country string `json:"country"`
			City    string `json:"city"`
			Number  string `json:"number"`
			Comment string `json:"comment"`
		} `json:"phones"`
	} `json:"contacts"`
}

func main() {
	// Параметры поиска
	searchText := "PHP"
	perPage := 100 // Максимум 100 вакансий на страницу

	fmt.Fprintln(os.Stderr, "[Статус] Получение вакансий PHP с hh.ru...")
	fmt.Println("Получение вакансий PHP с hh.ru...")
	fmt.Println()

	// Получаем первую страницу
	fmt.Fprintln(os.Stderr, "[Статус] Запрос первой страницы...")
	vacancies, err := getVacancies(searchText, 0, perPage)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка при получении вакансий: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "[Статус] Первая страница получена")

	if len(vacancies.Items) == 0 {
		fmt.Println("Вакансии не найдены")
		return
	}

	fmt.Printf("Найдено вакансий: %d\n", vacancies.Found)
	fmt.Printf("Получено на первой странице: %d\n", len(vacancies.Items))
	fmt.Fprintln(os.Stderr, fmt.Sprintf("[Статус] Найдено вакансий: %d, страниц: %d", vacancies.Found, vacancies.Pages))
	fmt.Println()
	fmt.Println("Список вакансий:")
	fmt.Println("================================================================================")

	// Выводим вакансии с первой страницы
	counter := 1
	fmt.Fprintln(os.Stderr, "[Статус] Обработка вакансий с первой страницы...")
	for _, vacancy := range vacancies.Items {
		printVacancy(counter, vacancy)
		counter++
	}
	fmt.Fprintln(os.Stderr, fmt.Sprintf("[Статус] Обработано вакансий с первой страницы: %d", len(vacancies.Items)))

	// Получаем остальные страницы
	for page := 1; page < vacancies.Pages; page++ {
		// Задержка 200мс между запросами, чтобы не превысить лимит API
		time.Sleep(200 * time.Millisecond)

		fmt.Fprintln(os.Stderr, fmt.Sprintf("[Статус] Запрос страницы %d из %d...", page+1, vacancies.Pages))
		nextVacancies, err := getVacancies(searchText, page, perPage)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка при получении страницы %d: %v\n", page, err)
			continue
		}

		for _, vacancy := range nextVacancies.Items {
			printVacancy(counter, vacancy)
			counter++
		}
		fmt.Fprintln(os.Stderr, fmt.Sprintf("[Статус] Обработано вакансий: %d", counter-1))
	}
	fmt.Fprintln(os.Stderr, "[Статус] Завершено! Всего обработано вакансий: "+fmt.Sprint(counter-1))
}

func printVacancy(counter int, vacancy Vacancy) {
	fmt.Printf("%d. %s\n", counter, vacancy.Name)
	fmt.Printf("   Компания: %s\n", vacancy.Employer.Name)

	// Выводим месторасположение
	if vacancy.Area.Name != "" {
		fmt.Printf("   Местоположение: %s\n", vacancy.Area.Name)
	}

	fmt.Printf("   Ссылка: %s\n", vacancy.AlternateURL)

	// Получаем детальную информацию с контактами
	time.Sleep(100 * time.Millisecond) // Задержка перед запросом деталей
	details, err := getVacancyDetails(vacancy.ID)
	if err == nil {
		// Выводим зарплату, если указана
		if details.Salary != nil {
			salaryStr := "   Зарплата: "
			if details.Salary.From != nil && details.Salary.To != nil {
				salaryStr += fmt.Sprintf("%d - %d %s", *details.Salary.From, *details.Salary.To, details.Salary.Currency)
			} else if details.Salary.From != nil {
				salaryStr += fmt.Sprintf("от %d %s", *details.Salary.From, details.Salary.Currency)
			} else if details.Salary.To != nil {
				salaryStr += fmt.Sprintf("до %d %s", *details.Salary.To, details.Salary.Currency)
			}
			fmt.Println(salaryStr)
		}

		// Выводим контакты, если есть
		if details.Contacts.Name != "" {
			fmt.Printf("   Контакты:\n")
			if details.Contacts.Name != "" {
				fmt.Printf("      ФИО: %s\n", details.Contacts.Name)
			}
			if details.Contacts.Email != "" {
				fmt.Printf("      Email: %s\n", details.Contacts.Email)
			}
			for _, phone := range details.Contacts.Phones {
				phoneStr := fmt.Sprintf("+%s (%s) %s", phone.Country, phone.City, phone.Number)
				if phone.Comment != "" {
					phoneStr += fmt.Sprintf(" (%s)", phone.Comment)
				}
				fmt.Printf("      Телефон: %s\n", phoneStr)
			}
		}
	}

	fmt.Println()
}

func getVacancies(text string, page, perPage int) (*VacanciesResponse, error) {
	baseURL := "https://api.hh.ru/vacancies"

	params := url.Values{}
	params.Add("text", text)
	params.Add("page", fmt.Sprintf("%d", page))
	params.Add("per_page", fmt.Sprintf("%d", perPage))
	params.Add("area", "113") // 113 - Россия, можно изменить

	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, err
	}

	// Устанавливаем User-Agent, так как API hh.ru требует его
	req.Header.Set("User-Agent", "HH Vacancy Fetcher/1.0 (your@email.com)")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API вернул статус %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var vacanciesResp VacanciesResponse
	err = json.Unmarshal(body, &vacanciesResp)
	if err != nil {
		return nil, err
	}

	return &vacanciesResp, nil
}

func getVacancyDetails(vacancyID string) (*VacancyDetails, error) {
	fullURL := fmt.Sprintf("https://api.hh.ru/vacancies/%s", vacancyID)

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "HH Vacancy Fetcher/1.0 (your@email.com)")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API вернул статус %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var details VacancyDetails
	err = json.Unmarshal(body, &details)
	if err != nil {
		return nil, err
	}

	return &details, nil
}
