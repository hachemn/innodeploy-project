package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"time"
)

type WebhookRequest struct {
	Repo string `json:"repo"`
}

func main() {

	// ✅ root
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("InnoDeploy API running 🚀"))
	})

	// 🔥 webhook
	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodPost {
			http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
			return
		}

		var req WebhookRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		log.Println("🚀 Pipeline started for:", req.Repo)

		// 🧹 cleanup old project
		exec.Command("rm", "-rf", "project").Run()

		// 📥 clone repo
		cmd := exec.Command("git", "clone", req.Repo, "project")
		output, err := cmd.CombinedOutput()

		if err != nil {
			log.Println("❌ Clone error:", string(output))
			http.Error(w, "Clone failed", 500)
			return
		}

		log.Println("✅ Clone success")
		// 🔥 generate version FIRST
		version := fmt.Sprintf("v%d", time.Now().Unix())
		image := "nourhenhachem/innodeploy-app:" + version

		// 🐳 build docker image WITH version
		buildCmd := exec.Command("sh", "-c", fmt.Sprintf("cd project && docker build -t %s .", image))
		buildOutput, err := buildCmd.CombinedOutput()

		if err != nil {
			log.Println("❌ Docker build error:", string(buildOutput))
			http.Error(w, "Docker build failed", 500)
			return
		}

		log.Println("✅ Docker build success")

		// 📤 push image
		pushCmd := exec.Command("docker", "push", image)
		pushOutput, err := pushCmd.CombinedOutput()

		if err != nil {
			log.Println("❌ Push error:", string(pushOutput))
			http.Error(w, "Push failed", 500)
			return
		}

		log.Println("✅ Push success")

		// 📄 update deployment.yaml
		data, err := os.ReadFile("../innodeploy-gitops/k8s/deployment.yaml")
		if err != nil {
			log.Println("❌ Read YAML error:", err)
			http.Error(w, "YAML read failed", 500)
			return
		}

		updated := regexp.MustCompile(`image: .*`).ReplaceAllString(string(data), "image: "+image)

		err = os.WriteFile("../innodeploy-gitops/k8s/deployment.yaml", []byte(updated), 0644)
		if err != nil {
			log.Println("❌ Write YAML error:", err)
			http.Error(w, "YAML write failed", 500)
			return
		}

		log.Println("✅ YAML updated")

		// 📤 commit & push GitOps repo
		exec.Command("git", "-C", "../innodeploy-gitops", "add", ".").Run()
		exec.Command("git", "-C", "../innodeploy-gitops", "commit", "-m", "update image "+version).Run()
		exec.Command("git", "-C", "../innodeploy-gitops", "push").Run()

		log.Println("✅ GitOps repo updated")


		w.Write([]byte("Pipeline started 🚀"))
	})

	http.HandleFunc("/rollback", func(w http.ResponseWriter, r *http.Request) {

	log.Println("⏪ Rollback triggered")

	repoPath := "../innodeploy-gitops"

	// go to previous commit ON main branch
	cmd := exec.Command("git", "-C", repoPath, "reset", "--hard", "HEAD~1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("❌ Rollback error:", string(out))
		http.Error(w, "Rollback failed", 500)
		return
	}

	// push rollback
	pushCmd := exec.Command("git", "-C", repoPath, "push", "--force", "origin", "main")
	pushOut, err := pushCmd.CombinedOutput()
	if err != nil {
		log.Println("❌ Push rollback error:", string(pushOut))
		http.Error(w, "Push failed", 500)
		return
	}

	log.Println("✅ Rollback pushed")

	w.Write([]byte("Rollback done 🚀"))
})

	log.Println("Server running on :8080")
	http.ListenAndServe(":8080", nil)
}