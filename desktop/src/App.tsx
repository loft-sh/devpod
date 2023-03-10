import {
  Tab,
  TabList,
  Tabs,
} from "@chakra-ui/react"
import {
  BrowserRouter as Router,
  Link, Route,
  Routes
} from 'react-router-dom';
import { ProvidersTab } from "./views/Providers/Providers"
import { WorkspacesTab } from "./views/Workspaces/Workspaces"
import {Layout} from "./components/Layout/Layout";
import { Open } from "./views/Open/Open";

export function App() {
  return (
      <Router>
        <Routes>
          <Route path={"/open"} element={<Open />} />
          <Route path={"*"} element={
            <Layout>
              <Tabs>
                <TabList>
                  <Tab>
                    <Link to={"/"}>Workspaces</Link>
                  </Tab>
                  <Tab>
                    <Link to={"/providers"}>Providers</Link>
                  </Tab>
                </TabList>
              </Tabs>
              <Routes>
                <Route path="/" element={<WorkspacesTab />} />
                <Route path="/providers" element={<ProvidersTab />} />
              </Routes>
            </Layout>} />
        </Routes>
      </Router>
  )
}
