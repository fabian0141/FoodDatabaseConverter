package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

var toKeepNutrients = map[string]bool{
	"Energy (Atwater General Factors)":   true,
	"Protein":                            true,
	"Carbohydrate, by summation":         true,
	"Total lipid (fat)":                  true,
	"Fiber, total dietary":               true,
	"Total dietary fiber (AOAC 2011.25)": true,

	"Vitamin A, RAE":                 true,
	"Thiamin":                        true,
	"Riboflavin":                     true,
	"Pantothenic acid":               true,
	"Vitamin B-6":                    true,
	"Vitamin B-12":                   true,
	"Vitamin C, total ascorbic acid": true,
	"Vitamin E (alpha-tocopherol)":   true,
	"Vitamin K (Menaquinone-4)":      true,

	"Calcium, Ca":   true,
	"Iron, Fe":      true,
	"Magnesium, Mg": true,
	"Phosphorus, P": true,
	"Potassium, K":  true,
	"Zinc, Zn":      true,
}

// 22 fields
/*type Food struct {
	id   int
	name string

	calories int
	protein  int
	carbs    int
	fat      int
	fiber    int

	vitaminA   int
	vitaminB1  int
	vitaminB2  int
	vitaminB5  int
	vitaminB6  int
	vitaminB12 int
	vitaminC   int
	vitaminE   int
	vitaminK   int

	calcium   int
	iron      int
	magnesium int
	phospher  int
	potassium int
	zinc      int
}*/

func main() {
	file, err, jsonData := getFoodData()
	foodList := filterFoodList(jsonData)

	exportJSON(file, err, foodList)

	db := openDB()

	createDB(db)

	insertFood(db, foodList)

}

func getFoodData() (*os.File, error, map[string]interface{}) {
	file, err := os.Open("original-food.json")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var jsonData map[string]interface{}

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&jsonData); err != nil {
		log.Fatal(err)
	}
	return file, err, jsonData
}

func filterFoodList(jsonData map[string]interface{}) []interface{} {
	foodList := jsonData["FoundationFoods"].([]interface{})

	for _, value := range foodList {
		food := value.(map[string]interface{})
		deleteNotNeededFoodAttribs(food)

		nutrients := deleteNotNeededNutrientsAttribs(food)
		addNutrientsToFood(food, nutrients)
	}
	return foodList
}

func deleteNotNeededFoodAttribs(food map[string]interface{}) {
	delete(food, "foodClass")
	delete(food, "isHistoricalReference")
	delete(food, "ndbNumber")
	delete(food, "foodPortions")
	delete(food, "publicationDate")
	delete(food, "nutrientConversionFactors")
	delete(food, "dataType")
	delete(food, "foodCategory")
	delete(food, "foodAttributes")
	delete(food, "inputFoods")
	delete(food, "scientificName")
}

func deleteNotNeededNutrientsAttribs(food map[string]interface{}) []interface{} {
	nutrients := food["foodNutrients"].([]interface{})
	for i := 0; i < len(nutrients); i++ {
		nutrient := nutrients[i].(map[string]interface{})
		name := nutrient["nutrient"].(map[string]interface{})["name"].(string)

		var removed bool
		nutrients, i, removed = removeNutrients(name, nutrients, i, nutrient)
		if removed {
			continue
		}

		nutrient["name"] = name
		deleteNutrientAttribs(nutrient)
	}
	return nutrients
}

func deleteNutrientAttribs(nutrient map[string]interface{}) {
	delete(nutrient, "foodNutrientDerivation")
	delete(nutrient, "nutrient")
	delete(nutrient, "id")
	delete(nutrient, "type")
	delete(nutrient, "dataPoints")
	delete(nutrient, "max")
	delete(nutrient, "min")
	delete(nutrient, "median")
}

func removeNutrients(name string, nutrients []interface{}, i int, nutrient map[string]interface{}) ([]interface{}, int, bool) {
	if keep := toKeepNutrients[name]; !keep {
		nutrients = append(nutrients[:i], nutrients[i+1:]...)
		i--
		return nutrients, i, true
	}
	if _, ok := nutrient["amount"]; !ok {
		nutrients = append(nutrients[:i], nutrients[i+1:]...)
		i--
		return nutrients, i, true
	}
	return nutrients, i, false
}

func addNutrientsToFood(food map[string]interface{}, nutrients []interface{}) {
	delete(food, "foodNutrients")
	for _, value := range nutrients {
		nutri := value.(map[string]interface{})
		food[nutri["name"].(string)] = nutri["amount"].(float64)
	}
}

func exportJSON(file *os.File, err error, foodList []interface{}) {
	file, err = os.Create("food.json")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ")
	if err := encoder.Encode(foodList); err != nil {
		log.Fatal(err)
	}
}

func openDB() *sql.DB {
	db, err := sql.Open("sqlite3", "food.db")
	if err != nil {
		log.Println(err)
	}
	return db
}

func createDB(db *sql.DB) {
	statement, err := db.Prepare(`CREATE TABLE IF NOT EXISTS Food (id INTEGER PRIMARY KEY, name VARCHAR(64), 
		calories FLOAT, protein FLOAT, carbs FLOAT, fat FLOAT, fiber FLOAT, 
		vitaminA FLOAT, vitaminB1 FLOAT, vitaminB2 FLOAT, vitaminB5 FLOAT, vitaminB6 FLOAT, vitaminB12 FLOAT, vitaminC FLOAT, vitaminE FLOAT, vitaminK FLOAT, 
		calcium FLOAT, iron FLOAT, magnesium FLOAT, phospher FLOAT, potassium FLOAT, zinc FLOAT);`)
	if err != nil {
		log.Println("Error in creating table", err)
	} else {
		log.Println("Successfully created table Food!")
	}
	statement.Exec()
}

func insertFood(db *sql.DB, foodList []interface{}) {
	statement, err := db.Prepare(`INSERT INTO Food (id, name, calories, protein, carbs, fat, fiber, 
		vitaminA, vitaminB1, vitaminB2, vitaminB5, vitaminB6, vitaminB12, vitaminC, vitaminE, vitaminK, 
		calcium, iron, magnesium, phospher, potassium, zinc) VALUES 
		(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`)

	if err != nil {
		log.Println("Error inserting row", err)
	}

	for _, value := range foodList {
		food := value.(map[string]interface{})

		id := int(food["fdcId"].(float64))
		name := food["description"].(string)

		calories := getNutrientAmount(food, "Energy (Atwater General Factors)")
		protein := getNutrientAmount(food, "Protein")
		carbs := getNutrientAmount(food, "Carbohydrate, by summation")
		fat := getNutrientAmount(food, "Total lipid (fat)")
		fiber := getNutrientAmount(food, "Fiber, total dietary")

		vitaminA := getNutrientAmount(food, "Vitamin A, RAE")
		vitaminB1 := getNutrientAmount(food, "Thiamin")
		vitaminB2 := getNutrientAmount(food, "Riboflavin")
		vitaminB5 := getNutrientAmount(food, "Pantothenic acid")
		vitaminB6 := getNutrientAmount(food, "Vitamin B-6")
		vitaminB12 := getNutrientAmount(food, "Vitamin B-12")
		vitaminC := getNutrientAmount(food, "Vitamin C, total ascorbic acid")
		vitaminE := getNutrientAmount(food, "Vitamin E (alpha-tocopherol)")
		vitaminK := getNutrientAmount(food, "Vitamin K (Menaquinone-4)")

		calcium := getNutrientAmount(food, "Calcium, Ca")
		iron := getNutrientAmount(food, "Iron, Fe")
		magnesium := getNutrientAmount(food, "Magnesium, Mg")
		phospher := getNutrientAmount(food, "Phosphorus, P")
		potassium := getNutrientAmount(food, "Potassium, K")
		zinc := getNutrientAmount(food, "Zinc, Zn")

		statement.Exec(id, name, calories, protein, carbs, fat, fiber,
			vitaminA, vitaminB1, vitaminB2, vitaminB5, vitaminB6, vitaminB12, vitaminC, vitaminE, vitaminK,
			calcium, iron, magnesium, phospher, potassium, zinc)
	}

}

func getNutrientAmount(food map[string]interface{}, names ...string) float64 {
	for i := 0; i < len(names); i++ {
		nutrient := food[names[i]]
		if nutrient != nil {
			return nutrient.(float64)
		}
	}
	return 0
}
