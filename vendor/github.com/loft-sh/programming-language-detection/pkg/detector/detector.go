package detector

import (
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

var globalLimit int = 10000
var fileCount int = 0
var result fileList = fileList{}

type fileList struct {
	mu   sync.Mutex
	data []string
}

func (i *fileList) appendFile(input string) {
	i.mu.Lock()
	i.data = append(i.data, input)
	i.mu.Unlock()
}

var supportedLanguages = map[string]*regexp.Regexp{
	"JavaScript": regexp.MustCompile(`(.*?)\.js[ $]`),
	"TypeScript": regexp.MustCompile(`(.*?)\.ts[ $]`),
	"Python":     regexp.MustCompile(`(.*?)\.py[ $]`),
	"C":          regexp.MustCompile(`(.*?)\.c[ $]`),
	"Cpp":        regexp.MustCompile(`(.*?)\.cpp[ $]`),
	"DotNet":     regexp.MustCompile(`(.*?)\.cs[ $]`),
	"Go":         regexp.MustCompile(`(.*?)\.go[ $]`),
	"PHP":        regexp.MustCompile(`(.*?)\.php[ $]`),
	"Java":       regexp.MustCompile(`(.*?)\.java[ $]`),
	"Rust":       regexp.MustCompile(`(.*?)\.rs[ $]`),
	"Ruby":       regexp.MustCompile(`(.*?)\.rb[ $]`),
}

var vendorMatchers = []*regexp.Regexp{
	regexp.MustCompile(`(.*?)\.d\.ts$`),
	regexp.MustCompile(`(3rd|[Tt]hird)[-_]?[Pp]arty`),
	regexp.MustCompile(`([^\s]*)import\.(css|less|scss|styl)$`),
	regexp.MustCompile(`(\.|-)min\.(js|css)$`),
	regexp.MustCompile(`-vsdoc\.js$`),
	regexp.MustCompile(`BuddyBuildSDK\.framework`),
	regexp.MustCompile(`Carthage`),
	regexp.MustCompile(`Chart\.js$`),
	regexp.MustCompile(`Control\.FullScreen\.css`),
	regexp.MustCompile(`Control\.FullScreen\.js`),
	regexp.MustCompile(`Crashlytics\.framework`),
	regexp.MustCompile(`Fabric\.framework`),
	regexp.MustCompile(`Godeps`),
	regexp.MustCompile(`Jenkinsfile$`),
	regexp.MustCompile(`Leaflet\.Coordinates-\d+\.\d+\.\d+\.src\.js$`),
	regexp.MustCompile(`MathJax`),
	regexp.MustCompile(`MochiKit\.js$`),
	regexp.MustCompile(`RealmSwift\.framework`),
	regexp.MustCompile(`Realm\.framework`),
	regexp.MustCompile(`Sparkle`),
	regexp.MustCompile(`Vagrantfile$`),
	regexp.MustCompile(`[Vv]+endor`),
	regexp.MustCompile(`\.[Dd][Ss]_[Ss]tore$`),
	regexp.MustCompile(`\.gitattributes$`),
	regexp.MustCompile(`\.github`),
	regexp.MustCompile(`\.gitignore$`),
	regexp.MustCompile(`\.gitmodules$`),
	regexp.MustCompile(`\.gitpod\.Dockerfile$`),
	regexp.MustCompile(`\.google_apis`),
	regexp.MustCompile(`\.imageset`),
	regexp.MustCompile(`\.indent\.pro`),
	regexp.MustCompile(`\.intellisense\.js$`),
	regexp.MustCompile(`\.osx$`),
	regexp.MustCompile(`\.sublime-project`),
	regexp.MustCompile(`\.sublime-workspace`),
	regexp.MustCompile(`\.vscode`),
	regexp.MustCompile(`\.xctemplate`),
	regexp.MustCompile(`\.yarn`),
	regexp.MustCompile(`^[Dd]ependencies`),
	regexp.MustCompile(`^debian`),
	regexp.MustCompile(`^deps`),
	regexp.MustCompile(`^rebar$`),
	regexp.MustCompile(`_esy$`),
	regexp.MustCompile(`ace-builds`),
	regexp.MustCompile(`aclocal\.m4`),
	regexp.MustCompile(`activator$`),
	regexp.MustCompile(`activator\.bat$`),
	regexp.MustCompile(`admin_media`),
	regexp.MustCompile(`angular([^.]*)\.js$`),
	regexp.MustCompile(`animate\.(css|less|scss|styl)$`),
	regexp.MustCompile(`bootbox\.js`),
	regexp.MustCompile(`bootstrap-datepicker`),
	regexp.MustCompile(`bower_components`),
	regexp.MustCompile(`bulma\.(css|sass|scss)$`),
	regexp.MustCompile(`cache`),
	regexp.MustCompile(`ckeditor\.js$`),
	regexp.MustCompile(`config\.guess$`),
	regexp.MustCompile(`config\.sub$`),
	regexp.MustCompile(`configure$`),
	regexp.MustCompile(`controls\.js$`),
	regexp.MustCompile(`cordova([^.]*)\.js$`),
	regexp.MustCompile(`cordova\-\d\.\d(\.\d)?\.js$`),
	regexp.MustCompile(`cpplint\.py`),
	regexp.MustCompile(`custom\.bootstrap([^\s]*)(js|css|less|scss|styl)$`),
	regexp.MustCompile(`dist`),
	regexp.MustCompile(`dojo\.js$`),
	regexp.MustCompile(`dotnet-install\.(ps1|sh)$`),
	regexp.MustCompile(`dragdrop\.js$`),
	regexp.MustCompile(`effects\.js$`),
	regexp.MustCompile(`env`),
	regexp.MustCompile(`erlang\.mk`),
	regexp.MustCompile(`fabfile\.py$`),
	regexp.MustCompile(`font-?awesome\.(css|less|scss|styl)$`),
	regexp.MustCompile(`fontello(.*?)\.css$`),
	regexp.MustCompile(`foundation(\..*)?\.js$`),
	regexp.MustCompile(`foundation\.(css|less|scss|styl)$`),
	regexp.MustCompile(`fuelux\.js`),
	regexp.MustCompile(`gradlew$`),
	regexp.MustCompile(`gradlew\.bat$`),
	regexp.MustCompile(`html5shiv\.js$`),
	regexp.MustCompile(`jquery([^.]*)\.js$`),
	regexp.MustCompile(`jquery([^.]*)\.unobtrusive\-ajax\.js$`),
	regexp.MustCompile(`jquery([^.]*)\.validate(\.unobtrusive)?\.js$`),
	regexp.MustCompile(`jquery\-\d\.\d+(\.\d+)?\.js$`),
	regexp.MustCompile(`jquery\-ui(\-\d\.\d+(\.\d+)?)?(\.\w+)?\.(js|css)$`),
	regexp.MustCompile(`jquery\.(ui|effects)\.([^.]*)\.(js|css)$`),
	regexp.MustCompile(`jquery\.dataTables\.js`),
	regexp.MustCompile(`jquery\.fancybox\.(js|css)`),
	regexp.MustCompile(`jquery\.fileupload(-\w+)?\.js$`),
	regexp.MustCompile(`jquery\.fn\.gantt\.js`),
	regexp.MustCompile(`knockout-(\d+\.){3}(debug\.)?js$`),
	regexp.MustCompile(`leaflet\.draw-src\.js`),
	regexp.MustCompile(`leaflet\.draw\.css`),
	regexp.MustCompile(`leaflet\.spin\.js`),
	regexp.MustCompile(`libtool\.m4`),
	regexp.MustCompile(`ltoptions\.m4`),
	regexp.MustCompile(`ltsugar\.m4`),
	regexp.MustCompile(`ltversion\.m4`),
	regexp.MustCompile(`lt~obsolete\.m4`),
	regexp.MustCompile(`materialize\.(css|less|scss|styl|js)$`),
	regexp.MustCompile(`modernizr\-\d\.\d+(\.\d+)?\.js$`),
	regexp.MustCompile(`modernizr\.custom\.\d+\.js$`),
	regexp.MustCompile(`mootools([^.]*)\d+\.\d+.\d+([^.]*)\.js$`),
	regexp.MustCompile(`mvnw$`),
	regexp.MustCompile(`mvnw\.cmd$`),
	regexp.MustCompile(`node_modules`),
	regexp.MustCompile(`normalize\.(css|less|scss|styl)$`),
	regexp.MustCompile(`octicons\.css`),
	regexp.MustCompile(`pdf\.worker\.js`),
	regexp.MustCompile(`proguard-rules\.pro$`),
	regexp.MustCompile(`proguard\.pro$`),
	regexp.MustCompile(`prototype(.*)\.js$`),
	regexp.MustCompile(`puphpet`),
	regexp.MustCompile(`react(-[^.]*)?\.js$`),
	regexp.MustCompile(`run\.n$`),
	regexp.MustCompile(`shBrush([^.]*)\.js$`),
	regexp.MustCompile(`shCore\.js$`),
	regexp.MustCompile(`shLegacy\.js$`),
	regexp.MustCompile(`skeleton\.(css|less|scss|styl)$`),
	regexp.MustCompile(`slick\.\w+.js$`),
	regexp.MustCompile(`sprockets-octicons\.scss`),
	regexp.MustCompile(`testdata`),
	regexp.MustCompile(`tiny_mce([^.]*)\.js$`),
	regexp.MustCompile(`vendors?`),
	regexp.MustCompile(`vignettes`),
	regexp.MustCompile(`waf$`),
	regexp.MustCompile(`wicket-leaflet\.js`),
	regexp.MustCompile(`yahoo-([^.]*)\.js$`),
	regexp.MustCompile(`yui([^.]*)\.js$`),
}

var documentationMatchers = []*regexp.Regexp{
	regexp.MustCompile(`CHANGE(S|LOG)?(\.|$)`),
	regexp.MustCompile(`CITATION(\.cff|(S)?(\.(bib|md))?)$`),
	regexp.MustCompile(`CODE_OF_CONDUCT(\.|$)`),
	regexp.MustCompile(`CONTRIBUTING(\.|$)`),
	regexp.MustCompile(`COPYING(\.|$)`),
	regexp.MustCompile(`INSTALL(\.|$)`),
	regexp.MustCompile(`LICEN[CS]E(\.|$)`),
	regexp.MustCompile(`MAINTAINERS(\.|$)`),
	regexp.MustCompile(`PULL_REQUEST_TEMPLATE(\.|$)`),
	regexp.MustCompile(`README(\.|$)`),
	regexp.MustCompile(`ROADMAP(\.|$)`),
	regexp.MustCompile(`SECURITY(\.|$)`),
	regexp.MustCompile(`[Dd]ocumentation`),
	regexp.MustCompile(`[Gg]roovydoc`),
	regexp.MustCompile(`[Jj]avadoc`),
	regexp.MustCompile(`[Ll]icen[cs]e(\.|$)`),
	regexp.MustCompile(`[Rr]eadme(\.|$)`),
	regexp.MustCompile(`^[Dd]emos?`),
	regexp.MustCompile(`^[Dd]ocs?`),
	regexp.MustCompile(`^[Ee]xamples`),
	regexp.MustCompile(`^[Mm]an`),
	regexp.MustCompile(`^[Ss]amples?`),
}

var configExtensions = map[string]struct{}{
	".xml":  {},
	".json": {},
	".toml": {},
	".yaml": {},
	".yml":  {},
	".conf": {},
	".ini":  {},
}

// IsConfiguration returns if the file or directory in input is a configuration file or not.
func IsConfiguration(input string) bool {
	extension := filepath.Ext(input)
	_, matched := configExtensions[extension]
	return matched
}

// IsDotFile returns if the file or directory in input is hidden or not.
func IsDotFile(input string) bool {
	return strings.HasPrefix(input, ".") && input != "."
}

// IsDocumentation returns if the file or directory in input is documentation related.
// This function will iter through a series of regexes to infere if that is the case.
func IsDocumentation(input string) bool {
	for _, matcher := range documentationMatchers {
		if matcher == nil {
			continue
		}
		if matcher.MatchString(input) {
			return true
		}
	}
	return false
}

// IsVendor returns if the file or directory in input is vendor related.
// This function will iter through a series of regexes to infere if that is the case.
func IsVendor(input string) bool {
	for _, matcher := range vendorMatchers {
		if matcher == nil {
			continue
		}
		if matcher.MatchString(input) {
			return true
		}
	}
	return false
}

// crawl will recursively explore the filesystem starting from input.
// it will parallelly explore each directory, stopping when either all files are
// explored or when we reach a globalLimit of file explored.
func crawl(input string, wg *sync.WaitGroup) {
	defer wg.Done()

	// we shouldn't exceed the globalLimit
	if fileCount > globalLimit {
		return
	}

	files, err := ioutil.ReadDir(input)
	if err != nil {
		return
	}

	// parallel cycle to explore the dirs
	for _, file := range files {
		// we should excloude configs, dotfiles, docs and vendors
		if IsConfiguration(file.Name()) ||
			IsDotFile(file.Name()) ||
			IsDocumentation(file.Name()) ||
			IsVendor(file.Name()) {

			continue
		} else if file.IsDir() {
			// in case of dirs, we spawn a new routine to start exploring
			// parallelly
			wg.Add(1)
			go crawl(input+"/"+file.Name(), wg)
		} else {
			// if instead it is a file, we add it to the list of found files
			// and if we didn't exceed the fileLimit, we continue the cycle
			fileCount++
			result.appendFile(input + "/" + file.Name())
			if fileCount > globalLimit {
				return
			}
		}

	}
	return
}

// GetLanguage will return a guess of which language is the project in the input
// path.
// It will return None if it didn't match anything, else it will return one of:
//
//	"JavaScript"
//	"TypeScript"
//	"Python"
//	"C"
//	"Cpp"
//	"DotNet"
//	"Go"
//	"PHP"
//	"Java"
//	"Rust"
//	"Ruby"
func GetLanguage(path string, limit int) string {
	wg := &sync.WaitGroup{}

	if limit > 0 {
		globalLimit = limit
	}
	fileCount = 0
	result = fileList{}

	wg.Add(1)
	go crawl(path, wg)
	wg.Wait()

	// Merge the results in a single line
	// so we can do a simple matchall and count the
	// matches
	output := strings.Join(result.data[:], " ") + " "
	lang := "None"
	max := 0
	for key, matcher := range supportedLanguages {
		if matcher == nil {
			continue
		}
		res := matcher.FindAllString(output, -1)
		if len(res) > max {
			lang = key
			max = len(res)
		}
	}

	// Return the language with the most occurrencies
	return lang
}
