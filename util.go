package main

import (
    "github.com/radovskyb/watcher"
    "encoding/json"
    "io/ioutil"
    "runtime"
    "strconv"
    "strings"
    "syscall"
    "bufio"
    "time"
    "fmt"
    "os"
)

func bToMb(b uint64) uint64 {
    return b / 1024 / 1024
}

func readInput(prompt string) string {
     fmt.Printf("%s %s", INFO, prompt)
     scanner := bufio.NewScanner(os.Stdin)

     for scanner.Scan() {
         return strings.TrimSpace(scanner.Text())
     }

     if err := scanner.Err(); err != nil {
         fmt.Printf("%s Error reading input - %s\n", ERROR, err.Error())
         os.Exit(1)
     }

     return ""
 }

func formatNumber(number int64) string {
    in := strconv.FormatInt(number, 10)
    out := make([]byte, len(in) + (len(in) - 2 + int(in[0] / '0')) / 3)

    if in[0] == '-' {
        in, out[0] = in[1:], '-'
    }

    for i, j, k := len(in) - 1, len(out) - 1, 0; ; i, j = i - 1, j - 1 {
        out[j] = in[i]

        if i == 0 {
            return string(out)
        }

        if k++; k == 3 {
            j, k = j - 1, 0
            out[j] = ','
        }
    }
}

func loadConfig() configStruct {
    file, err := ioutil.ReadFile("./data/config.json")

    if err != nil {
        fmt.Printf("%s %s\n", ERROR, err.Error()); os.Exit(1)
    }
    
    var configData configStruct

    err = json.Unmarshal(file, &configData)

    if err != nil {
        fmt.Printf("%s %s\n", ERROR, err.Error())
        os.Exit(1)
    }

    if len(configData.Discord.APIVersion) == 0 { apiVersion = "6" } else { apiVersion = configData.Discord.APIVersion }

    if len(configData.Discord.Webhooks.EmbedMedia) > 5 {
        discordEmbedPicture = configData.Discord.Webhooks.EmbedMedia
    } else {
        discordEmbedPicture = "https://i.imgur.com/E4E2DNK.png"
    }

    return configData
}

func checkDataFolderExists() {
    if _, err := os.Stat("./data"); os.IsNotExist(err) {
        folderCreationErr := os.Mkdir("data", os.ModePerm);
        if folderCreationErr != nil { return }

        createFile("./data/alts.txt")
        createFile("./data/claimToken.txt")

        _, creationErr := os.Create("./data/config.json")
        if creationErr != nil { return }

        var mainClaimToken string = readInput("Discord Claim Token: ")
        var successWebhook string = readInput("Discord Success Webhook: ")
        var missedWebhook string = readInput("Discord Missed Webhook: ")

        var configData configStruct

        configData.Discord.APIVersion = "9"
        configData.Discord.Webhooks.Successful = successWebhook
        configData.Discord.Webhooks.Missed = missedWebhook

        configData.Sniper.SaveInvites = false
        configData.Sniper.SnipeOnMain = false
        configData.Sniper.Threads = 10

        jsonBytes, _ := json.MarshalIndent(configData, "", " ")
        jsonWriteErr := ioutil.WriteFile("./data/config.json", jsonBytes, 0644)
        if jsonWriteErr != nil { return }

        claimTokenFile, fileOpenErr := os.OpenFile("./data/claimToken.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
        if fileOpenErr != nil { return }
        _, writeError := claimTokenFile.WriteString(mainClaimToken)
        if (writeError) != nil { return };
        claimTokenFile.Close()

        fmt.Printf("\n%s Please go to /data/ and put your alts into alts.txt\n", ERROR)
        os.Exit(0)
    }
}

func createFile(fileName string) {
    file, err := os.Create(fileName)

    if err != nil { return }
    defer file.Close()
}

func readFile(path string) (lines []string) {
    file, err := os.Open(path)

    if err != nil {
        fmt.Printf("%s %s\n", ERROR, err.Error())
        os.Exit(1)
    }

    defer file.Close()
    var scanner *bufio.Scanner = bufio.NewScanner(file)

    for scanner.Scan() {
        if len(scanner.Text()) > 3 {
            lines = append(lines, strings.TrimSpace(scanner.Text()))
        }
    }

    if err := scanner.Err(); err != nil {
        fmt.Printf("%s Error reading line - %s\n", ERROR, err.Error())
    }

    return
}

func readSingleFile(path string) (line string) {
    data, err := ioutil.ReadFile(path)

    if err != nil {
        fmt.Printf("%s %s\n", ERROR, err.Error())
        os.Exit(1)
    }

    return string(strings.TrimSpace(string(data)))
}

func useAllThreads() {
    config.Sniper.Threads = runtime.NumCPU()
}

func sortAlts() {
    for i := 0; i < config.Sniper.Threads; i++ {
        splitAlts = append(splitAlts, alts[i * len(alts) / config.Sniper.Threads : (i + 1) * len(alts) / config.Sniper.Threads])
    }
}

func watchConfigChanges() {
    w := watcher.New()

    go func() {
        for {
            select {
            case event := <-w.Event:
                _ = event
                reloadConfig("./data/config.json")
            case err := <-w.Error:
                fmt.Println(err)
            case <-w.Closed:
                return
            }
        }
    }()

    if err := w.Add("./data/config.json"); err != nil { return }
    go func() { w.Wait() }()
    if err := w.Start(time.Millisecond * 1000); err != nil { return }
}

func watchTokenChanges() {
    w := watcher.New()

    go func() {
        for {
            select {
            case event := <-w.Event:
                _ = event

                claimToken = readSingleFile("./data/claimToken.txt")
                    
                buildClaimHeaders()
            case err := <-w.Error:
                fmt.Println(err)
            case <-w.Closed:
                return
            }
        }
    }()

    if err := w.Add("./data/claimToken.txt"); err != nil { return }
    go func() { w.Wait() }()
    if err := w.Start(time.Millisecond * 1000); err != nil { return }
}

func reloadConfig(path string) {
    file, err := ioutil.ReadFile(path)

    if err != nil { return }
    var configData configStruct
    err = json.Unmarshal(file, &configData)
    if err != nil { return }

    if len(configData.Discord.APIVersion) == 0 { apiVersion = "6" } else { apiVersion = configData.Discord.APIVersion }

    if len(configData.Discord.Webhooks.EmbedMedia) > 5 {
        discordEmbedPicture = configData.Discord.Webhooks.EmbedMedia
    } else {
        discordEmbedPicture = "https://i.imgur.com/E4E2DNK.png"
    }
}

func setULimit() bool {
    var rLimit syscall.Rlimit

    rLimit.Cur = 999999
    rLimit.Max = 999999

    return syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit) == nil
}

func alreadyClaimedNitro(nitro string) bool {
    for _, nitros := range claimed {
        if nitros == nitro {
            return true
        }
    }

    return false
}

func convertMilliseconds(milliseconds time.Duration) string {
    return fmt.Sprintf("%f", float64(milliseconds) / float64(time.Second))
}

func saveInvites() {
    if !config.Sniper.SaveInvites { return }
    if len(invites) < 100 { return }

    if _, existsErr := os.Stat("./data/invites.txt"); os.IsNotExist(existsErr) {
        _, creationErr := os.Create("./data/invites.txt")
        if creationErr != nil { return }

        file, fileOpenErr := os.OpenFile("./data/invites.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
        if fileOpenErr != nil { return }

        for _, invite := range invites {
            _, writeError := file.WriteString("discord.gg/" + invite + "\n")

            _ = writeError
        }

        file.Close()
    } else {
        file, fileOpenErr := os.OpenFile("./data/invites.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
        if fileOpenErr != nil { return }

        for _, invite := range invites {
            _, writeError := file.WriteString("discord.gg/" + invite + "\n")

            _ = writeError
        }

        file.Close()
    }

    invites = nil
}