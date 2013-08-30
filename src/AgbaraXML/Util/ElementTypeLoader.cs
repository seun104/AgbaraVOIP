using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Reflection;
using Emmanuel.AgbaraVOIP.AgbaraXML;

namespace Emmanuel.AgbaraVOIP.AgbaraXML.Utils
{
    public class ElementTypeLoader
    {
        public static void ScanForElements(Assembly assembly, Action<string, Type> foundAction)
        {
            Type elementType = typeof(Element);
            foreach (Type type in assembly.GetTypes())
            {
                if (elementType.IsAssignableFrom(type))
                {
                    foundAction(type.Name, type);
                }

            }
        }
    }
}
