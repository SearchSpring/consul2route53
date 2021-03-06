package consul2route53

import(
	"strconv"
	"log"
)

type Consul2Route53 struct {
	*Config 
	*Consul 
	*Route53Srv
}

func New(c Config) *Consul2Route53 {
	ret := &Consul2Route53{
		Config: &c,
		Consul: &Consul{Config: &c},
		Route53Srv: &Route53Srv{Config: &c},
	}

	return ret
}

func (c *Consul2Route53) Run() error {
	// Get services from consul
	log.Print("Getting services from consul")
	err := c.Consul.GetServices()
	if err != nil {
		return err
	}
	servicesmap := c.ServicesMap()

	// Get records from Route53
	log.Print("Getting records from route53")

	recordsmap, err := c.RecordsMap()
	if err != nil {
		return err
	}
	// Create changes
	recordchanges := make(map[string][]Record)
	for service := range servicesmap {
		domain := c.Zone()
		// A records
		aname := service+"."+domain
		arecord := Record{
			Type: "A",
			Name: aname,
			Value: servicesmap[service].Address,
		}
		if exists := recordsmap[aname+"_A"]; exists != nil {
			if arecord != *recordsmap[aname+"_A"] {
				recordchanges["UPSERT"] = append(recordchanges["UPSERT"],arecord)
				delete(recordsmap,service+"_A")
			}
		} else {
			recordchanges["CREATE"] = append(recordchanges["CREATE"],arecord)
		} 
		// SRV records
		port := strconv.FormatInt(servicesmap[service].Port,10)
		srvname := "_"+service+"._tcp."+domain 
		srvrecord := Record{
			Type: "SRV",
			Name: srvname,
			Value: "1 1 "+port+" "+aname,
		}
		if exists := recordsmap[srvname+"_SRV"]; exists != nil {
			if srvrecord != *recordsmap[srvname+"_SRV"] {
				recordchanges["UPSERT"] = append(recordchanges["UPSERT"],srvrecord)
				delete(recordsmap,srvname+"_SRV")
			}
		} else {
			recordchanges["CREATE"] = append(recordchanges["CREATE"],srvrecord)
		}
	} 
	for changetype := range recordchanges {
		for _,record := range recordchanges[changetype] {
			c.AddChange(changetype,record)
		}
	}
	// Execute changes
	log.Printf("Committing %d changes",c.ChangesNum())
	if c.ChangesNum() > 0 {
		err = c.ChangeRecords()
		if err != nil {
			return err
		}
	} 
	return nil
}